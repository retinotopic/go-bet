package lobby

import (
	"bytes"
	"io"

	"github.com/Nerdmaster/poker"
	json "github.com/bytedance/sonic"
	// "github.com/coder/websocket"
)

type Ctrl struct {
	Place int    `json:"Place"`
	Ctrl  int    `json:"Ctrl"`
	Text  string `json:"Text"`
	Plr   int    `json:"-"` //index of player in AllUsers
	Type  string `json:"-"`
}

type PlayUnit struct {
	Cards            poker.CardList     `json:"Cards"`
	IsFold           bool               `json:"IsFold"`
	IsAway           bool               `json:"IsAway"`
	HasActed         bool               `json:"HasActed"`
	Bank             int                `json:"Bank"`
	Bet              int                `json:"Bet"`
	Place            int                `json:"Place"`
	TimeTurn         int64              `json:"TimeTurn"` // turn time in seconds
	Name             string             `json:"Name"`
	User_id          string             `json:"UserId"`
	SidePots         [8]int             `json:"-"`
	Conn             io.ReadWriteCloser `json:"-"`
	cache            *bytes.Buffer      `json:"-"`
	Encoder          json.Encoder       `json:"-"`
	HiddenCardsCache []byte             `json:"-"`
}

func (p *PlayUnit) StoreCache() {
	var err error

	err = p.Encoder.Encode(p)
	if err != nil {
		panic(err)
	}
	if len(p.HiddenCardsCache) < p.cache.Len() {
		p.HiddenCardsCache = make([]byte, p.cache.Len())
	}
	HideCards(p.HiddenCardsCache, p.cache.Bytes())
}

type GameBoard struct {
	Cards       poker.CardList `json:"Cards"`
	Bank        int            `json:"Bank"`
	MaxBet      int            `json:"MaxBet"`
	TurnPlace   int            `json:"TurnPlace"`
	DealerPlace int            `json:"DealerPlace"`
	Deadline    int64          `json:"Deadline"`
	Blind       int            `json:"Blind"`
	Active      bool           `json:"Active"`
	IsRating    bool           `json:"IsRating"`
	cache       *bytes.Buffer  `json:"-"`
	Encoder     json.Encoder   `json:"-"`
	CardsEval   poker.CardList `json:"-"`
}

func (g *GameBoard) StoreCache() {
	var err error

	err = g.Encoder.Encode(g)
	if err != nil {
		panic(err)
	}
}
