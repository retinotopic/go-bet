package lobby

import (
	"github.com/chehsunliu/poker"
	"github.com/coder/websocket"
	"github.com/goccy/go-json"
)

type GameBoard struct {
	Bank        int          `json:"Bank"`
	TurnPlace   int          `json:"TurnPlace"`
	DealerPlace int          `json:"DealerPlace"`
	Deadline    int64        `json:"Deadline"`
	Blind       int          `json:"Blind"`
	Cards       []poker.Card `json:"Cards"`
}

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
	Conn     *websocket.Conn `json:"-"`
	Name     string          `json:"Name"`
	User_id  string          `json:"UserId"`
}

func (pu *PlayUnit) UnmarshalJSON(data []byte) error {
	type Alias PlayUnit
	aux := &struct {
		Cards []json.RawMessage `json:"Cards"`
		*Alias
	}{
		Alias: (*Alias)(pu),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	pu.Cards = make([]poker.Card, len(aux.Cards))
	for i, rawCard := range aux.Cards {
		if err := json.Unmarshal(rawCard, &pu.Cards[i]); err != nil {
			return err
		}
	}

	return nil
}

func (pu *PlayUnit) MarshalJSON() ([]byte, error) {
	type Alias PlayUnit
	return json.Marshal(&struct {
		Cards []string `json:"Cards"`
		*Alias
	}{
		Cards: func() []string {
			cards := make([]string, len(pu.Cards))
			for i, card := range pu.Cards {
				cards[i] = card.String()
			}
			return cards
		}(),
		Alias: (*Alias)(pu),
	})
}
