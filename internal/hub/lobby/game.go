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
	Board       *PlayUnit   //
	SmallBlind  int         //
	MaxBet      int         //
	TurnTimer   *time.Timer //
	BlindTimer  *time.Timer //
	StartGameCh chan bool
	stop        chan bool
}

func (g *Game) tillTurnOrBlind() {
	for {
		select {
		case <-g.TurnTimer.C:
			idx := g.PlayersRing.Idx
			g.Players[idx].IsAway = true
			g.PlayerCh <- Ctrl{Plr: g.Players[idx]}
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
			var stages int
			g.stop = make(chan bool, 1)
			randsrc := rand.NewSource(int64(new(maphash.Hash).Sum64()))
			rand.New(randsrc).Shuffle(len(g.PlayersRing.Players), func(i, j int) {
				g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
			})
			g.DealNewHand()
			g.TurnTimer = time.NewTimer(time.Second * 30)
			g.BlindTimer = time.NewTimer(time.Minute * 5)
			go g.tillTurnOrBlind()
			defer func() { // prevent goroutine leakage
				g.stop <- true
			}()
			for GAME_LOOP := true; GAME_LOOP; {
				select {
				case ctrl, ok := <-g.PlayerCh:
					if ok && ctrl.Plr.Place == g.PlayersRing.Idx {
						g.TurnTimer.Stop()
						if ctrl.CtrlBet <= ctrl.Plr.Bankroll && ctrl.CtrlBet >= 0 {
							ctrl.Plr.CtrlBet = ctrl.CtrlBet
						} else {
							ctrl.Plr.CtrlBet = 0
						}
						pl := ctrl.Plr
						pl.DeadlineTurn = 0
						pl.Bet = pl.Bet + pl.CtrlBet
						pl.Bankroll -= pl.CtrlBet

						if pl.Bet >= g.MaxBet {
							g.MaxBet = pl.Bet
						} else {
							pl.IsFold = true
						}
						if pl.IsAway {
							pl.TimeTurn = 7
						}
						g.BroadcastCh <- pl

						pl.HasActed = true

						pl = g.Players[g.PlayersRing.Next(1)]
						for pl.IsFold && pl.IsAway {
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
									GAME_LOOP = false
									break
								}
								stages = -1
							}
							for _, v := range g.Players {
								v.HasActed, v.IsFold = false, false
								if !v.IsAway && v.TimeTurn < 20 {
									v.TimeTurn += 5
								}
							}
							stages++
						}
						dur := time.Duration(pl.TimeTurn)
						pl.DeadlineTurn = time.Now().Add(time.Second * dur).Unix()
						g.TurnTimer.Reset(time.Second * dur)
						g.BroadcastCh <- pl
					}
				case <-g.Shutdown:
					GAME_LOOP = false
					break
				}
			}
			g.ValidateCh <- Ctrl{Plr: g.Board}
		case <-g.PlayerCh:
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
	g.Players[g.PlayersRing.Next(1)].ValueSec = time.Now().Add(time.Second * 30).Unix()
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
