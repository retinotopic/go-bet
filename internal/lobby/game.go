package lobby

import (
	"math"
	"sort"
	"time"

	"github.com/chehsunliu/poker"
	"github.com/retinotopic/go-bet/internal/queue"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

type Game struct {
	*Lobby
	Deck       *poker.Deck
	Board      PlayUnit     //
	SmallBlind int          //
	MaxBet     int          //
	TurnTicker *time.Ticker //
	BlindTimer *time.Timer  //
}

func (g *Game) timerTillBlind() {
	g.BlindTimer = time.NewTimer(time.Minute * 5)
	for range g.BlindTimer.C {
		g.SmallBlind *= 2
		g.BlindTimer.Reset(time.Minute * 5)
	}
}
func (g *Game) tickerTillNextTurn() {
	g.TurnTicker = time.NewTicker(time.Second * 1)
	timevalue := 0
	for range g.TurnTicker.C {
		timevalue++
		g.PlayerBroadcast <- g.Players[g.PlayersRing.Idx].SendTimeValue(timevalue)
		if timevalue >= 30 {
			g.TurnTicker.Stop()
			g.PlayerCh <- *g.Players[g.PlayersRing.Idx]
		}
	}
}
func (g *Game) Game() {
	var stages int
	randfuncs.NewSource().Shuffle(g.LenPlayers, func(i, j int) {
		g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
	})
	g.PlayerBroadcast = make(chan PlayUnit)
	g.DealNewHand()
	go g.tickerTillNextTurn()
	go g.timerTillBlind()
	for pl := range g.PlayerCh { // evaluating hand
		if pl.Place == g.PlayersRing.Idx {
			g.TurnTicker.Stop()
			g.Lock()
			if pl.HasActed {
				continue
			}
			pl.HasActed = true
			g.Unlock()
			if pl.Bet >= g.MaxBet {
				g.MaxBet = pl.Bet
			} else {
				pl.IsFold = true
			}
			for pl.IsFold {
				pl = *g.PlayersRing.Next(1)
			}
			if pl.Bet == g.MaxBet && pl.HasActed {
				switch turn {
				case flop: // preflop to flop
					flopcard := g.Deck.Draw(3)
					g.Board.Cards = append(g.Board.Cards, flopcard...)
				case turn, river: // flop to turn, turn to river
					flopcard := g.Deck.Draw(1)
					g.Board.Cards = append(g.Board.Cards, flopcard[0])
				case postriver:
					g.PostRiver()
					stages = -1
				}
				for _, v := range g.Players {
					v.HasActed = false
				}
			}
			stages++
			go g.tickerTillNextTurn()
		}
	}
}
func (g *Game) PostRiver() {
	g.CalcWinners()
	newPlayers := make([]*PlayUnit, 0, 8)
	elims := make([]*PlayUnit, 0, 8)
	var last int
	for i, v := range g.Players {
		if v.Bankroll > 0 {
			newPlayers = append(newPlayers, v)
		} else {
			last = len(g.Players) - i
			elims = append(elims, v)
		}
	}
	if len(elims) != 0 {
		g.CalcRating(elims, last)
	}
	if len(newPlayers) == 1 {
		g.CalcRating(newPlayers, 1)
		return
	}
	g.Players = newPlayers
	g.DealNewHand()
}

func (g *Game) DealNewHand() {
	g.Deck = poker.NewDeck()
	g.Deck.Shuffle()

	for i, v := range g.Players {
		v.Cards = g.Deck.Draw(2)
		v.Place = i
	}
	g.Board = PlayUnit{}
	g.Board.Cards = make([]poker.Card, 0, 7)

	g.PlayersRing.Next(1).Bet = g.SmallBlind
	g.PlayersRing.Next(2).Bet = g.SmallBlind * 2
	//send all players to all players (broadcast)
	for _, pl := range g.Players {
		g.PlayerBroadcast <- *pl
	}
	g.TurnTicker.Reset(time.Second * 30)
}

func (g *Game) CalcWinners() {
	var emptyCard poker.Card
	topPlaces := make([]top, 0, 7)
	g.Board.Cards = append(g.Board.Cards, emptyCard, emptyCard)
	for i, v := range g.Players {
		if !v.IsFold {
			g.Board.Cards[5] = v.Cards[0]
			g.Board.Cards[6] = v.Cards[1]
			eval := poker.Evaluate(g.Board.Cards)
			topPlaces = append(topPlaces, top{rating: eval, place: i})
		}
		v.IsFold = false
	}
	sort.Slice(topPlaces, func(i, j int) bool {
		return topPlaces[i].rating > topPlaces[j].rating
	})
	i := 0
	for ; i == 0 || topPlaces[i].rating == topPlaces[i-1].rating; i++ {
	}
	share := g.Board.Bankroll / i
	for pl := range i {
		g.Players[topPlaces[pl].place].Bankroll += share
	}
}
func (g *Game) CalcRating(plr []*PlayUnit, place int) {
	baseChange := 30
	middlePlace := float64(g.LenPlayers+1) / 2
	for _, plr := range plr {
		rating := int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
		data, err := queue.NewMessage(plr.User_id, rating)
		if err != nil {
			queue.Queue.PublishTask(data)
		}
	}
}
