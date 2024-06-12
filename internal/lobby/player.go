package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/fasthttp/websocket"
)

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
	NextPlayer   *PlayUnit
	CurrentLobby *Lobby
}

func (p PlayUnit) PrivateSend() PlayUnit {
	return PlayUnit{Cards: []poker.Card{}, Place: p.Place}
}
func (p PlayUnit) SendTimeValue(time int) PlayUnit {
	return PlayUnit{ValueSec: time, Place: p.Place}
}
