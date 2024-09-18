package lobby

import (
	"hash/maphash"
	"math/rand"
	"sort"
	"time"

	"github.com/chehsunliu/poker"
)

type Game struct {
	*Lobby
	Deck        *poker.Deck
	Board       *PlayUnit    //
	SmallBlind  int          //
	MaxBet      int          //
	TurnTicker  *time.Ticker //
	BlindTimer  *time.Timer  //
	StartGameCh chan bool
	stop        chan bool
}

func (g *Game) tillTurnOrBlind() {
	timevalue := 0
	for {
		select {
		case <-g.TurnTicker.C:
			timevalue++
			idx := g.PlayersRing.Idx
			g.Players[idx].ValueSec = timevalue
			g.BroadcastCh <- g.Players[idx]
			if timevalue >= g.Players[idx].ExpirySec {
				g.Players[idx].ExpirySec /= 2
				g.TurnTicker.Stop()
				g.PlayerCh <- g.Players[idx]
			}
		case <-g.BlindTimer.C:
			g.SmallBlind *= 2
			g.BlindTimer.Reset(time.Minute * 5)
		case <-g.stop:
			return
		}
	}
}
func (g *Game) Game() {
	for {
		select {
		case <-g.StartGameCh:
			g.StartGame()
			g.ValidateCh <- Ctrl{plr: g.Board}
		case <-g.PlayerCh:
		case <-g.Shutdown:
			return
		}
	}

}
func (g *Game) StartGame() {
	for len(g.PlayerCh) > 0 {
		<-g.PlayerCh
	}
	var stages int
	g.stop = make(chan bool, 1)
	randsrc := rand.NewSource(int64(new(maphash.Hash).Sum64()))
	rand.New(randsrc).Shuffle(len(g.PlayersRing.Players), func(i, j int) {
		g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
	})
	g.DealNewHand()

	g.TurnTicker = time.NewTicker(time.Second * 1)
	g.BlindTimer = time.NewTimer(time.Minute * 5)
	go g.tillTurnOrBlind()
	defer func() { // prevent goroutine leakage
		g.stop <- true
	}()
	for {
		select {
		case pl, ok := <-g.PlayerCh:
			if pl.Place == g.PlayersRing.Idx && ok {
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
		case <-g.Shutdown:
			return
		}
	}

}
func (g *Game) PostRiver() (gameOver bool) {
	g.CalcWinners()
	newPlayers := make([]*PlayUnit, 0, 8)
	elims := make([]*PlayUnit, 0, 8)
	for _, v := range g.Players {
		if v.Bankroll > 0 {
			newPlayers = append(newPlayers, v)
		} else {
			elims = append(elims, v)
		}
	}
	if len(newPlayers) == 1 {
		g.PlayerOut(newPlayers)
		return true
	} else if len(elims) != 0 {
		g.PlayerOut(elims)
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
	topPlaces := make([]top, 0, 8)
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
	for j := range i {
		pl := g.Players[topPlaces[j].place]
		pl.Bankroll += share
		pl.Exposed = true
		g.BroadcastCh <- pl
		pl.Exposed = false
	}
	time.Sleep(time.Second * 4)
}
