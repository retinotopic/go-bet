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
	User_id      int
	Name         string `json:"Name"`
	Bankroll     int    `json:"Bankroll"`
	Bet          int    `json:"Bet"`
	IsFold       bool   `json:"IsFold"`
	HasActed     bool   `json:"HasActed"`
	Conn         *websocket.Conn
	Place        int          `json:"Place"`
	Admin        bool         `json:"IsAdmin"`
	Cards        []poker.Card `json:"Hand,omitempty"`
	ValueSec     int          `json:"Time,omitempty"`
	CurrentLobby *Lobby
}

func (p PlayUnit) PrivateSend() PlayUnit {
	return PlayUnit{Cards: []poker.Card{}, Place: p.Place}
}
func (p PlayUnit) SendTimeValue(time int) PlayUnit {
	return PlayUnit{ValueSec: time, Place: p.Place}
}
