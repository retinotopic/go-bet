package lobby

import (
	"sync"

	json "github.com/bytedance/sonic"
	"github.com/chehsunliu/poker"
	"github.com/coder/websocket"
)

type PlayersRing struct {
	Players  []*PlayUnit
	Idx      int
	Board    GameBoard
	Blindlvl int
}

// small blind is calculated via: initial player stack * stackShare[i]
var stackShare = []float32{
	0.01,
	0.02,
	0.03,
	0.05,
	0.075,
	0.1,
	0.15,
	0.20,
	0.25,
	0.3}

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
	Cards    []poker.Card    `json:"Cards"`
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
		panic("somehow bytedance/sonic messed up badly...")
	}
	return p.cache
}

type GameBoard struct {
	Bank        int          `json:"Bank"`
	TurnPlace   int          `json:"TurnPlace"`
	DealerPlace int          `json:"DealerPlace"`
	Deadline    int64        `json:"Deadline"`
	Blind       int          `json:"Blind"`
	Cards       []poker.Card `json:"Cards"`
	mtx         sync.RWMutex `json:"-"`
	cache       []byte       `json:"-"`
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
