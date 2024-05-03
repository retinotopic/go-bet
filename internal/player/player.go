package player

import (
	"github.com/retinotopic/go-bet/pkg/deck"

	"github.com/gorilla/websocket"
)

func NewPlayer() *Player {
	return &Player{}
}

type Player struct {
	Name     string `json:"Name"`
	Bankroll int    `json:"Bankroll"`
	Conn     *websocket.Conn
	Place    int          `json:"Place"`
	Admin    bool         `json:"IsAdmin"`
	Hand     [2]deck.Card `json:"Hand,omitempty"`
	ValueSec int          `json:"Time,omitempty"`
}

func (p Player) PrivateSend() Player {
	p.Hand = [2]deck.Card{}
	return p
}
func (p Player) SendTimeValue(time int) Player {
	return Player{ValueSec: time, Place: p.Place}
}
