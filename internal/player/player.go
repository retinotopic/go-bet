package player

import (
	"github.com/retinotopic/go-bet/pkg/deck"

	"github.com/gorilla/websocket"
)

type Player struct {
	Name     string `json:"Name"`
	Bankroll int    `json:"Bankroll"`
	Bet      int    `json:"Bet"`
	IsFold   bool   `json:"IsFold"`
	Conn     *websocket.Conn
	Place    int         `json:"Place"`
	Admin    bool        `json:"IsAdmin"`
	Cards    []deck.Card `json:"Hand,omitempty"`
	ValueSec int         `json:"Time,omitempty"`
}

func (p Player) PrivateSend() Player {
	return Player{Cards: []deck.Card{}, Place: p.Place}
}
func (p Player) SendTimeValue(time int) Player {
	return Player{ValueSec: time, Place: p.Place}
}
