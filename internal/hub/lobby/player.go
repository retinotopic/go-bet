package lobby

import (
	"sync"

	"github.com/Nerdmaster/poker"
	json "github.com/bytedance/sonic"
	"github.com/coder/websocket"
)

type MapUsers struct {
	M   map[string]*PlayUnit // id to user/player
	Mtx sync.RWMutex
}

type PlayersRing struct {
	Players  []*PlayUnit
	AllUsers MapUsers
	Idx      int
	Board    GameBoard
	Blindlvl int
}

func (rs *PlayersRing) Next(offset int) *PlayUnit {
	pl := rs.Players[rs.Idx]
	for pl.IsAway || pl.IsFold {
		rs.Idx = (rs.Idx + offset) % len(rs.Players)
		pl = rs.Players[rs.Idx]
	}
	return pl
}
func (rs *PlayersRing) NextDealer(start, offset int) int {
	rs.Idx = (start + offset) % len(rs.Players)
	return rs.Idx
}

type PlayUnit struct {
	Cards    []string        `json:"Cards"`
	IsFold   bool            `json:"IsFold"`
	IsAway   bool            `json:"IsAway"`
	HasActed bool            `json:"HasActed"`
	Bank     int             `json:"Bank"`
	Bet      int             `json:"Bet"`
	Place    int             `json:"Place"`
	TimeTurn int64           `json:"TimeTurn"` // turn time in seconds
	Name     string          `json:"Name"`
	User_id  string          `json:"UserId"`
	Conn     *websocket.Conn `json:"-"`
	mtx      sync.RWMutex    `json:"-"`
	cache    []byte          `json:"-"`
	cards    poker.CardList  `json:"-"`
}

func (p *PlayUnit) GetCache() []byte {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.cache
}
func (p *PlayUnit) StoreCache() []byte {
	p.mtx.Lock()
	defer p.mtx.Unlock()
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
	TurnPlace   int            `json:"TurnPlace"`
	DealerPlace int            `json:"DealerPlace"`
	Deadline    int64          `json:"Deadline"`
	Blind       int            `json:"Blind"`
	mtx         sync.RWMutex   `json:"-"`
	cache       []byte         `json:"-"`
	cards       poker.CardList `json:"-"`
	HiddenCards []string       `json:"-"`
}

func (g *GameBoard) GetCache() []byte {
	g.mtx.RLock()
	defer g.mtx.RUnlock()
	return g.cache
}
func (g *GameBoard) StoreCache() []byte {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	var err error
	g.cache, err = json.Marshal(g)
	if err != nil {
		panic("somehow bytedance/sonic messed up badly...")
	}
	return g.cache
}
