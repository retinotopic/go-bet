package lobby

import (
// "sync"
)

func (rs *Lobby) Next(offset int) int {
	pl := &rs.AllUsers[rs.Players[rs.Idx]]
	for pl.IsAway || pl.IsFold || pl.IsAllIn {
		rs.Idx = (rs.Idx + offset) % len(rs.Players)
	}
	return rs.Players[rs.Idx]
}
func (rs *Lobby) NextDealer(start, offset int) int {
	rs.Idx = (start + offset) % len(rs.Players)
	return rs.Idx
}

// small blind is calculated via: initial player stack * stackShare[i]
var stackShare = []float32{
	0.01,
	0.02,
	0.03,
	0.05,
	0.075,
	0.1,
	0.15,
	0.20,
	0.25,
	0.3}
