package lobby

import (
	"github.com/Nerdmaster/poker"
	json "github.com/bytedance/sonic"
	"github.com/coder/websocket"
)

type Ctrl struct {
	Place int    `json:"Place"`
	Ctrl  int    `json:"Ctrl"`
	Text  string `json:"Text"`
	Plr   int    `json:"-"` //index of player in AllUsers
}

type PlayUnit struct {
	Cards     []string        `json:"Cards"`
	IsFold    bool            `json:"IsFold"`
	IsAway    bool            `json:"IsAway"`
	HasActed  bool            `json:"HasActed"`
	Bank      int             `json:"Bank"`
	Bet       int             `json:"Bet"`
	Place     int             `json:"Place"`
	TimeTurn  int64           `json:"TimeTurn"` // turn time in seconds
	Name      string          `json:"Name"`
	User_id   string          `json:"UserId"`
	IsAllIn   bool            `json:"-"`
	Conn      *websocket.Conn `json:"-"`
	cache     []byte          `json:"-"`
	CardsEval poker.CardList  `json:"-"`
}

func (p *PlayUnit) StoreCache() []byte {
	var err error
	p.cache, err = json.Marshal(p)
	if err != nil {
		panic("somehow bytedance messed up badly...")
	}
	return p.cache
}

type GameBoard struct {
	Cards       []string       `json:"Cards"`
	Bank        int            `json:"Bank"`
	MaxBet      int            `json:"MaxBet"`
	TurnPlace   int            `json:"TurnPlace"`
	DealerPlace int            `json:"DealerPlace"`
	Deadline    int64          `json:"Deadline"`
	Blind       int            `json:"Blind"`
	Active      bool           `json:"Active"`
	IsRating    bool           `json:"IsRating"`
	cache       []byte         `json:"-"`
	Cardlist    poker.CardList `json:"-"`
	HiddenCards []string       `json:"-"`
}

func (g *GameBoard) StoreCache() []byte {
	var err error
	g.cache, err = json.Marshal(g)
	if err != nil {
		panic("somehow bytedance/sonic messed up badly...")
	}
	return g.cache
}
