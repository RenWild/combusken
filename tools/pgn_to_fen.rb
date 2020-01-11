# frozen_string_literal: true

require 'parallel'
require 'pgn'

@q = Queue.new
reader = Thread.new do
  acc = +''
  IO.foreach('./games.pgn') do |line|
    next if line.start_with?('{White')
    if !acc.empty? && line.include?('Event')
      @q << acc
      acc = +line
    else
      acc << line
    end
  end
  @q << acc
  @q << Parallel::Stop
  @q.close
end

@result = File.new('./games.fen', 'a')

Parallel.each(@q, in_processes: 7) do |pgn|
  game = nil
  begin
    game = PGN.parse(pgn).first
  rescue Exception => e
    p e
    puts pgn
    puts "---"
    next
  end
  game.moves.each_with_index do |move, idx|
    break if move.comment.nil?
    next if idx.zero? || move.comment.include?('book') || move.comment.include?('+0.00/1')# do not include position from book
    break if move.comment.include?('M') || move.comment.include?('#') # stop if mate was found in position

    @result.write(game.positions[idx].to_fen.to_s + ";#{game.result}\n")
    @result.flush
  end
end
reader.join
