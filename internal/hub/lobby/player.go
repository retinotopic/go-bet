package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/fasthttp/websocket"
)

type PlayersRing struct {
	Players []PlayUnit
	Idx     int
}

func (rs *PlayersRing) Next(offset int) int {
	rs.Idx = (rs.Idx + offset) % len(rs.Players)
	return rs.Idx
}

type PlayUnit struct {
	Exposed   bool `json:"Exposed,omitempty"`
	IsFold    bool `json:"IsFold,omitempty"`
	HasActed  bool `json:"HasActed,omitempty"`
	Admin     bool `json:"IsAdmin,omitempty"`
	IsRating  bool `json:"IsRating,omitempty"`
	Rating    int  `json:"Rating,omitempty"`
	Bankroll  int  `json:"Bankroll,omitempty"`
	Bet       int  `json:"Bet,omitempty"`
	CtrlBet   int  `json:"CtrlBet,omitempty"`
	Place     int  `json:"Place"`
	ValueSec  int  `json:"Time,omitempty"`
	ExpirySec int
	Conn      *websocket.Conn
	URLlobby  uint64
	Cards     []poker.Card `json:"Hand,omitempty"`
	Name      string       `json:"Name,omitempty"`
	Message   string       `json:"Message,omitempty"`
	User_id   string       `json:"UserId,omitempty"`
}

func (p PlayUnit) PrivateSend() PlayUnit {
	return PlayUnit{Place: p.Place}
}
func (p PlayUnit) SendTimeValue(time int) PlayUnit {
	return PlayUnit{ValueSec: time, Place: p.Place}
}
