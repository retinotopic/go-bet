package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/fasthttp/websocket"
)

type GameBoard struct {
	Bank          int          `json:"Bank"`
	TillNextBlind int          `json:"TillNextBlind"`
	TurnPlace     int          `json:"TurnPlace"`
	DealerPlace   int          `json:"DealerPlace"`
	DeadlineTurn  int64        `json:"DeadlineTurn"` // deadline date (unix seconds)
	CurrentBlind  int          `json:"CurrentBlind"`
	Cards         []poker.Card `json:"Cards"`
}

type PlayersRing struct {
	Players []*PlayUnit
	Idx     int
	Board   GameBoard
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
