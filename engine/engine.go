package engine

import "context"
import "errors"
import "runtime"
import "github.com/mhib/combusken/backend"
import "github.com/mhib/combusken/evaluation"

const MAX_HEIGHT = 127
const STACK_SIZE = MAX_HEIGHT + 1

var errTimeout = errors.New("Search timeout")

type IntUciOption struct {
	Name string
	Min  int
	Max  int
	Val  int
}

type TransTable interface {
	Get(key uint64, height int) (ok bool, value int16, depth int16, move backend.Move, flag uint8)
	Set(key uint64, value, depth int, move backend.Move, flag, height int)
	Clear()
}

type Engine struct {
	Hash         IntUciOption
	Threads      IntUciOption
	MoveOverhead IntUciOption
	PawnHash     IntUciOption
	done         <-chan struct{}
	TransTable
	evaluation.PawnKingTable
	RepeatedPositions map[uint64]interface{}
	MovesCount        int
	Update            func(SearchInfo)
	timeManager
	threads []thread
}

type thread struct {
	engine *Engine
	MoveEvaluator
	nodes int
	stack [STACK_SIZE]StackEntry
}

type UciScore struct {
	Mate      int
	Centipawn int
}

func newUciScore(score int) UciScore {
	if score >= ValueWin {
		return UciScore{Mate: (evaluation.Mate - score + 1) / 2}
	} else if score <= ValueLoss {
		return UciScore{Mate: (-evaluation.Mate - score) / 2}
	} else {
		return UciScore{Centipawn: score}
	}
}

type SearchInfo struct {
	Score UciScore
	Depth int
	Nodes int
	Moves []backend.Move
}

type StackEntry struct {
	position backend.Position
	PV
	moves                [256]backend.EvaledMove
	quietsSearched       [256]backend.Move
	evaluation           int16
	evaluationCalculated bool
}

func (se *StackEntry) invalidateEvaluation() {
	se.evaluationCalculated = false
}

func (se *StackEntry) Evaluation(pk evaluation.PawnKingTable) int16 {
	if !se.evaluationCalculated {
		se.evaluation = int16(evaluation.Evaluate(&se.position, pk))
		se.evaluationCalculated = true
	}
	return se.evaluation
}

func (se *StackEntry) NonCachedEvaluation(pk evaluation.PawnKingTable) int {
	return evaluation.Evaluate(&se.position, pk)
}

type PV struct {
	size  int
	items [STACK_SIZE]backend.Move
}

type LimitsType struct {
	Ponder         bool
	Infinite       bool
	WhiteTime      int
	BlackTime      int
	WhiteIncrement int
	BlackIncrement int
	MoveTime       int
	MovesToGo      int
	Depth          int
	Nodes          int
	Mate           int
}

type SearchParams struct {
	Positions []backend.Position
	Limits    LimitsType
}

func (e *Engine) GetInfo() (name, version, author string) {
	return "Combusken", "0.0.2", "Marcin Henryk Bartkowiak"
}

func (e *Engine) GetOptions() []*IntUciOption {
	return []*IntUciOption{&e.Hash, &e.Threads, &e.PawnHash, &e.MoveOverhead}
}

func NewEngine() (ret Engine) {
	ret.Hash = IntUciOption{"Hash", 4, 2048, 256}
	ret.Threads = IntUciOption{"Threads", 1, runtime.NumCPU(), 1}
	ret.PawnHash = IntUciOption{"PawnHash", 0, 8, 2}
	ret.MoveOverhead = IntUciOption{"Move Overhead", 0, 10000, 50}
	ret.threads = make([]thread, 1)
	return
}

func (e *Engine) Search(ctx context.Context, searchParams SearchParams) backend.Move {
	e.fillMoveHistory(searchParams.Positions)
	e.timeManager = newTimeManager(searchParams.Limits, e.MoveOverhead.Val, searchParams.Positions[len(searchParams.Positions)-1].SideToMove)
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	if e.hardTimeout() > 0 {
		ctx, cancel = context.WithTimeout(ctx, e.hardTimeout())
	}
	defer cancel()
	e.done = ctx.Done()
	return e.bestMove(ctx, &searchParams.Positions[len(searchParams.Positions)-1])
}

func (e *Engine) fillMoveHistory(positions []backend.Position) {
	e.MovesCount = len(positions) - 1
	moveHistory := make(map[uint64]int)
	for i := len(positions) - 1; i >= 0; i-- {
		moveHistory[positions[i].Key]++
		if positions[i].FiftyMove == 0 {
			break
		}
	}
	e.RepeatedPositions = make(map[uint64]interface{})
	for key, count := range moveHistory {
		if count >= 2 {
			e.RepeatedPositions[key] = struct{}{}
		}
	}
}

func (e *Engine) NewGame() {
	if e.Threads.Val == 1 {
		e.TransTable = NewSingleThreadTransTable(e.Hash.Val)
	} else {
		e.TransTable = NewAtomicTransTable(e.Hash.Val)
	}
	e.threads = make([]thread, e.Threads.Val)
	for i := range e.threads {
		e.threads[i].MoveEvaluator = MoveEvaluator{}
		e.threads[i].engine = e
	}
	e.PawnKingTable = evaluation.NewPKTable(e.PawnHash.Val)
}

func (e *Engine) callUpdate(s SearchInfo) {
	if e.Update != nil {
		e.Update(s)
	}
}

func (e *Engine) nodes() (sum int) {
	for i := range e.threads {
		sum += e.threads[i].nodes
	}
	return
}

func (t *thread) incNodes() {
	t.nodes++
	if (t.nodes % 255) == 0 {
		select {
		case <-t.engine.done:
			panic(errTimeout)
		default:
		}
	}
}

func (t *thread) PawnKingTable() evaluation.PawnKingTable {
	return t.engine.PawnKingTable
}

func (pv *PV) clear() {
	pv.size = 0
}

func (pv *PV) assign(m backend.Move, child *PV) {
	pv.size = 1 + child.size
	pv.items[0] = m
	copy(pv.items[1:], child.Moves())
}

func (t *thread) isImproving(height int) bool {
	if !t.stack[height].evaluationCalculated {
		t.stack[height].evaluation = int16(evaluation.Evaluate(&t.stack[height].position, t.engine.PawnKingTable))
		t.stack[height].evaluationCalculated = true
	}
	if !t.stack[height-2].evaluationCalculated {
		t.stack[height-2].evaluation = int16(evaluation.Evaluate(&t.stack[height-2].position, t.engine.PawnKingTable))
		t.stack[height-2].evaluationCalculated = true
	}
	return t.stack[height].evaluation >= t.stack[height-2].evaluation
}

func (pv *PV) Moves() []backend.Move {
	return pv.items[:pv.size]
}
