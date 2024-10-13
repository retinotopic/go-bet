package lobby

import (
	"sync"
)

type MapUsers struct {
	M   map[string]*PlayUnit // userId to user || player
	Mtx sync.RWMutex
}

type PlayersRing struct {
	Players  []*PlayUnit
	AllUsers MapUsers
	Idx      int
	Board    GameBoard
	Blindlvl int
}

func (rs *PlayersRing) Next(offset int) *PlayUnit {
	pl := rs.Players[rs.Idx]
	for pl.IsAway || pl.IsFold {
		rs.Idx = (rs.Idx + offset) % len(rs.Players)
		pl = rs.Players[rs.Idx]
	}
	return pl
}
func (rs *PlayersRing) NextDealer(start, offset int) int {
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
