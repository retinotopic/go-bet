package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/fasthttp/websocket"
)

type PlayersRing struct {
	Players []*PlayUnit
	Idx     int
}

func (rs *PlayersRing) Next(offset int) int {
	pl := rs.Players[rs.Idx]
	for pl.IsAway || pl.IsFold {
		rs.Idx = (rs.Idx + offset) % len(rs.Players)
		pl = rs.Players[rs.Idx]
	}
	return rs.Idx
}
func (rs *PlayersRing) NextDealer(start, offset int) int {
	rs.Idx = (start + offset) % len(rs.Players)
	return rs.Idx
}

type PlayUnit struct {
	IsFold   bool `json:"IsFold"`
	IsAway   bool `json:"IsAway"`
	HasActed bool `json:"HasActed"`
	Bankroll int  `json:"Bankroll"`
	Bet      int  `json:"Bet"`
	/* -1 means the player is not at the table
	0 means the table of cards itself, not the player
	and anything greater than 0 and less than 9 is the players at the table*/
	Place        int             `json:"Place"`
	TimeTurn     int64           `json:"TimeTurn"`     // turn time in seconds
	DeadlineTurn int64           `json:"DeadlineTurn"` // deadline date (unix seconds)
	Conn         *websocket.Conn `json:"-"`
	Cards        []poker.Card    `json:"Hand"`
	Name         string          `json:"Name"`
	User_id      string          `json:"UserId"`
}
