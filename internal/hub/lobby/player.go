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
	Exposed  bool `json:"Exposed,omitempty"` //means whether the cards should be shown to everyone
	IsFold   bool `json:"IsFold,omitempty"`
	HasActed bool `json:"HasActed,omitempty"`
	Admin    bool `json:"IsAdmin,omitempty"`
	IsRating bool `json:"IsRating,omitempty"`
	Rating   int  `json:"Rating,omitempty"`
	Bankroll int  `json:"Bankroll,omitempty"`
	Bet      int  `json:"Bet,omitempty"`
	CtrlBet  int  `json:"CtrlBet,omitempty"`
	Place    int  `json:"Place"` //-1 means the player is not at the table
	//0 means the table of cards itself, not the player
	//and anything greater than 0 and less than 9 is the players at the table
	ValueSec  int `json:"Time,omitempty"`
	ExpirySec int
	Conn      *websocket.Conn
	URLlobby  uint64
	Cards     []poker.Card `json:"Hand,omitempty"`
	Name      string       `json:"Name,omitempty"`
	Message   string       `json:"Message,omitempty"`
	User_id   string       `json:"UserId,omitempty"`
}
