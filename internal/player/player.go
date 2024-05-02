package player

import (
	"github.com/retinotopic/pokerGO/pkg/deck"

	"github.com/gorilla/websocket"
)

func NewPlayer() *Player {
	return &Player{}
}

type Player struct {
	Name     string `json:"Name"`
	Bankroll int    `json:"Bankroll"`
	Bet      int    `json:"Bet"`
	Conn     *websocket.Conn
	Place    int       `json:"Place"`
	Admin    bool      `json:"IsAdmin"`
	Hand     deck.Card `json:"Hand,omitempty"`
	ValueSec int       `json:"Time,omitempty"`
}

func (p Player) PrivateSend() Player {
	p.Hand = deck.Card{}
	return p
}
func (p Player) SendTimeValue(time int) Player {
	return Player{ValueSec: time, Place: p.Place}
}
