package backend

import (
	"math/rand"
)

var zobrist [6][2][64]uint64
var zobristEpSquare [64]uint64
var zobristFlags [16]uint64
var zobristFifty [4]uint64
var zobristColor uint64

func initZobrist() {
	var r = rand.New(rand.NewSource(0))
	for y := 0; y < 6; y++ {
		for x := 0; x < 2; x++ {
			for z := 0; z < 64; z++ {
				zobrist[y][x][z] = r.Uint64()
			}
		}
	}
	for y := 24; y <= 39; y++ {
		zobristEpSquare[y] = r.Uint64()
	}
	for y := 0; y < 16; y++ {
		zobristFlags[y] = r.Uint64()
	}
	for y := 0; y < 4; y++ {
		zobristFifty[y] = r.Uint64()
	}
	zobristColor = r.Uint64()
}

func HashPosition(pos *Position) {
	pos.Key = 0
	var fromId int
	var fromBB uint64

	for fromBB = pos.Pawns & pos.White; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[0][0][fromId]
	}
	for fromBB = pos.Knights & pos.White; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[1][0][fromId]
	}
	for fromBB = pos.Bishops & pos.White; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[2][0][fromId]
	}
	for fromBB = pos.Rooks & pos.White; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[3][0][fromId]
	}
	for fromBB = pos.Queens & pos.White; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[4][0][fromId]
	}
	for fromBB = pos.Kings & pos.White; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[5][0][fromId]
	}

	for fromBB = pos.Pawns & pos.Black; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[0][1][fromId]
	}
	for fromBB = pos.Knights & pos.Black; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[1][1][fromId]
	}
	for fromBB = pos.Bishops & pos.Black; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[2][1][fromId]
	}
	for fromBB = pos.Rooks & pos.Black; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[3][1][fromId]
	}
	for fromBB = pos.Queens & pos.Black; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[4][1][fromId]
	}
	for fromBB = pos.Kings & pos.Black; fromBB != 0; fromBB &= (fromBB - 1) {
		fromId = BitScan(fromBB)
		pos.Key ^= zobrist[5][1][fromId]
	}
	pos.Key ^= zobristFlags[pos.Flags]
	if pos.WhiteMove {
		pos.Key ^= zobristColor
	}
	pos.Key ^= zobristEpSquare[pos.EpSquare]
	pos.Key ^= zobristFifty[pos.FiftyMove&3]
}

func init() {
	initZobrist()
}
