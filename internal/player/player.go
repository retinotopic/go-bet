package player

import (
	"github.com/chehsunliu/poker"
	"github.com/gorilla/websocket"
)

type PlayUnit struct {
	Name       string `json:"Name"`
	Bankroll   int    `json:"Bankroll"`
	Bet        int    `json:"Bet"`
	IsFold     bool   `json:"IsFold"`
	HasActed   bool   `json:"HasActed"`
	Conn       *websocket.Conn
	Place      int          `json:"Place"`
	Admin      bool         `json:"IsAdmin"`
	Cards      []poker.Card `json:"Hand,omitempty"`
	ValueSec   int          `json:"Time,omitempty"`
	NextPlayer *PlayUnit
}

func (p PlayUnit) PrivateSend() PlayUnit {
	return PlayUnit{Cards: []poker.Card{}, Place: p.Place}
}
func (p PlayUnit) SendTimeValue(time int) PlayUnit {
	return PlayUnit{ValueSec: time, Place: p.Place}
}
