package lobby

import (
	"hash/maphash"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/chehsunliu/poker"
)

type Game struct {
	*Lobby
	Deck       *poker.Deck
	Board      PlayUnit     //
	SmallBlind int          //
	MaxBet     int          //
	TurnTicker *time.Ticker //
	BlindTimer *time.Timer  //
	stop       chan bool
}

func (g *Game) timerTillBlind() {
	for {
		select {
		case <-g.BlindTimer.C:
			g.SmallBlind *= 2
			g.BlindTimer.Reset(time.Minute * 5)
		case <-g.stop:
			return
		}
	}
}
func (g *Game) tickerTillNextTurn() {
	timevalue := 0
	for {
		select {
		case <-g.TurnTicker.C:
			timevalue++
			idx := g.PlayersRing.Idx
			g.BroadcastCh <- g.Players[idx].SendTimeValue(timevalue)
			if timevalue >= g.Players[idx].ExpirySec {
				g.Players[idx].ExpirySec /= 2
				g.TurnTicker.Stop()
				g.PlayerCh <- g.Players[idx]
			}
		case <-g.stop:
			return
		}
	}
}
func (g *Game) Game() {
	var stages int
	g.stop = make(chan bool, 2)
	rand.New(rand.NewSource(int64(new(maphash.Hash).Sum64()))).Shuffle(g.LenPlayers, func(i, j int) {
		g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
	})
	g.DealNewHand()

	g.TurnTicker = time.NewTicker(time.Second * 1)
	go g.tickerTillNextTurn()
	g.BlindTimer = time.NewTimer(time.Minute * 5)
	go g.timerTillBlind()
	defer func() { // prevent goroutine leakage
		g.stop <- true
		g.stop <- true
	}()
	for pl := range g.PlayerCh { // evaluating hand
		if pl.Place == g.PlayersRing.Idx {
			g.TurnTicker.Stop()

			pl.Bet = pl.Bet + pl.CtrlBet
			pl.Bankroll -= pl.CtrlBet

			if pl.Bet >= g.MaxBet {
				g.MaxBet = pl.Bet
			} else {
				pl.IsFold = true
			}

			g.BroadcastCh <- pl

			pl.HasActed = true
			g.PlayersRing.Next(1)
			for pl.IsFold {
				pl = g.Players[g.PlayersRing.Next(1)]
			}
			if pl.Bet == g.MaxBet && pl.HasActed {
				switch stages {
				case flop: // preflop to flop
					flopcard := g.Deck.Draw(3)
					g.Board.Cards = append(g.Board.Cards, flopcard...)
					g.BroadcastCh <- g.Board
				case turn, river: // flop to turn, turn to river
					flopcard := g.Deck.Draw(1)
					g.Board.Cards = append(g.Board.Cards, flopcard[0])
					g.BroadcastCh <- g.Board
				case postriver:
					if ok := g.PostRiver(); ok {
						return
					}
					stages = -1
				}
				for _, v := range g.Players {
					v.HasActed, v.IsFold = false, false
				}
				stages++
			}

			g.TurnTicker.Reset(time.Second * 30)
		}
	}
}
func (g *Game) PostRiver() (gameOver bool) {
	g.CalcWinners()
	newPlayers := make([]PlayUnit, 0, 8)
	elims := make([]PlayUnit, 0, 8)
	var last int
	for i, v := range g.Players {
		if v.Bankroll > 0 {
			newPlayers = append(newPlayers, v)
		} else {
			last = len(g.Players) - i
			elims = append(elims, v)
		}
	}
	if len(newPlayers) == 1 {
		g.CalcRating(newPlayers, 1)
		return true
	} else if len(elims) != 0 {
		g.CalcRating(elims, last)
		g.Players = newPlayers
		g.Idx -= len(elims)
	}
	g.DealNewHand()
	return
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

	g.Players[g.PlayersRing.Next(1)].Bet = g.SmallBlind
	g.Players[g.PlayersRing.Next(1)].Bet = g.SmallBlind * 2
	g.PlayersRing.Next(1)
	//send all players to all players (broadcast)
	for _, pl := range g.Players {
		g.BroadcastCh <- pl
	}
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
func (g *Game) CalcRating(plr []PlayUnit, place int) {
	if !g.IsRating {
		return
	}
	baseChange := 30
	middlePlace := float64(g.LenPlayers+1) / 2
	for _, plr := range plr {
		rating := int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
		data, err := g.queue.NewMessage(plr.User_id, rating)
		if err != nil {
			g.queue.PublishTask(data, 5)
		}
		plr.Conn.WriteJSON(plr)
		plr.Conn.Close()
	}
}
