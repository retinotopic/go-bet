package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/fasthttp/websocket"
)

type PlayersRing struct {
	Players []*PlayUnit
	Idx     int
}

func (rs *PlayersRing) Next(offset int) *PlayUnit {
	rs.Idx = (rs.Idx + offset) % len(rs.Players)
	return rs.Players[rs.Idx]
}

type PlayUnit struct {
	User_id   int    `json:"UserId,omitempty"`
	IsRating  bool   `json:"IsRating,omitempty"`
	Rating    int    `json:"Rating,omitempty"`
	Name      string `json:"Name,omitempty"`
	Bankroll  int    `json:"Bankroll,omitempty"`
	Bet       int    `json:"Bet,omitempty"`
	IsFold    bool   `json:"IsFold,omitempty"`
	HasActed  bool   `json:"HasActed,omitempty"`
	Conn      *websocket.Conn
	Place     int          `json:"Place"`
	Admin     bool         `json:"IsAdmin,omitempty"`
	Cards     []poker.Card `json:"Hand,omitempty"`
	ValueSec  int          `json:"Time,omitempty"`
	ExpirySec int
	URLlobby  uint64
}

func (p PlayUnit) PrivateSend() PlayUnit {
	return PlayUnit{Place: p.Place}
}
func (p PlayUnit) SendTimeValue(time int) PlayUnit {
	return PlayUnit{ValueSec: time, Place: p.Place}
}
