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
	rs.Idx = (rs.Idx + offset) % len(rs.Players)
	return rs.Idx
}

type PlayUnit struct {
	IsExposed bool `json:"Exposed"` //means whether the cards should be shown to everyone
	IsFold    bool `json:"IsFold"`
	IsAway    bool `json:"IsAway"`
	HasActed  bool `json:"HasActed"`
	Bankroll  int  `json:"Bankroll"`
	Bet       int  `json:"Bet"`
	CtrlBet   int  `json:"CtrlBet"`
	/* -1 means the player is not at the table
	0 means the table of cards itself, not the player
	and anything greater than 0 and less than 9 is the players at the table*/
	Place        int   `json:"Place"`
	DeadlineTurn int64 `json:"DeadlineTurn"` // deadline date (unix seconds)
	TimeTurn     int64 `json:"TimeTurn"`     // turn time in seconds
	Conn         *websocket.Conn
	URLlobby     uint64
	Cards        []poker.Card `json:"Hand,omitempty"`
	Name         string       `json:"Name,omitempty"`
	User_id      string       `json:"UserId,omitempty"`
}
