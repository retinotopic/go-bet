package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/coder/websocket"
)

type GameBoard struct {
	Bank        int          `json:"Bank"`
	TurnPlace   int          `json:"TurnPlace"`
	DealerPlace int          `json:"DealerPlace"`
	Deadline    int64        `json:"Deadline"`
	Blind       int          `json:"Blind"`
	Cards       []poker.Card `json:"Cards"`
}

type PlayersRing struct {
	Players  []*PlayUnit
	Idx      int
	Board    GameBoard
	Blindlvl int
}

// blind is calculated via: initial player stack * stackShare[ i ]
var stackShare = []float32{
	0.01,
	0.02,
	0.03,
	0.05,
	0.075,
	0.1,
	0.15,
	0.25,
	0.3}

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

type PlayUnit struct {
	IsFold   bool            `json:"IsFold"`
	IsAway   bool            `json:"IsAway"`
	HasActed bool            `json:"HasActed"`
	Bank     int             `json:"Bank"`
	Bet      int             `json:"Bet"`
	Place    int             `json:"Place"`
	TimeTurn int64           `json:"TimeTurn"` // turn time in seconds
	Conn     *websocket.Conn `json:"-"`
	Cards    []poker.Card    `json:"Cards"`
	Name     string          `json:"Name"`
	User_id  string          `json:"UserId"`
}
