package lobby

import (
	"hash/maphash"
	"math/rand"
	"sort"
	"time"

	"github.com/Nerdmaster/poker"
)

type top struct {
	place  int
	rating int32
}
type Implementor interface {
	Validate(Ctrl)
	PlayerOut([]*PlayUnit, int)
}
type Game struct {
	*Lobby
	Deck        *Deck
	MaxBet      int         //
	TurnTimer   *time.Timer //
	BlindTimer  *time.Timer //
	StartGameCh chan bool
	stop        chan bool
	Impl        Implementor
}

func (g *Game) tillTurnOrBlind() {
	for {
		select {
		case <-g.TurnTimer.C:
			idx := g.PlayersRing.Idx
			g.Players[idx].IsAway = true
			g.PlayerCh <- Ctrl{Plr: g.Players[idx]}
		case <-g.BlindTimer.C:
			if len(stackShare) > g.Blindlvl {
				g.Blindlvl++
				g.Board.Blind *= int(stackShare[g.Blindlvl])
				g.BlindTimer.Reset(time.Minute * 5)
			} else {
				g.BlindTimer.Stop()
			}
		case <-g.stop:
			return
		}
	}
}

func (g *Game) Game() {
	for {
		select {
		case <-g.StartGameCh:
			var pl *PlayUnit
			var stages int
			var now int64
			for _, v := range g.Players { // give the player his initial bank
				v.Bank = g.Board.Bank
			}
			g.Board.DealerPlace = 0

			g.Board.HiddenCards = make([]string, 5)
			g.Board.cards = make([]poker.Card, 7)
			g.Board.Cards = make([]string, 0, 5)

			g.stop = make(chan bool, 1)
			randsrc := rand.NewSource(int64(new(maphash.Hash).Sum64()))
			rand.New(randsrc).Shuffle(len(g.PlayersRing.Players), func(i, j int) { // shuffle player seats
				g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
			})
			g.DealNewHand()
			g.TurnTimer = time.NewTimer(time.Second * 10)
			g.BlindTimer = time.NewTimer(time.Duration(g.Board.Deadline) * time.Minute)
			go g.tillTurnOrBlind()
			defer func() { // prevent goroutine leakage
				g.stop <- true
			}()
			for GAME_LOOP := true; GAME_LOOP; {
				select {
				case ctrl, ok := <-g.PlayerCh:
					if ok && ctrl.Plr.Place == g.PlayersRing.Idx {

						g.TurnTimer.Stop()
						now = time.Now().Unix()
						pl = ctrl.Plr
						if ctrl.CtrlBet > pl.Bank || ctrl.CtrlBet < 0 { // if a player tries to cheat the system or just wants to quit, then we zero his bank
							pl.Bank = 0
							ctrl.CtrlBet = 0
						}

						if pl.Bet+ctrl.CtrlBet >= g.MaxBet {
							g.MaxBet = pl.Bet
							pl.IsAway = false
							pl.Bet = pl.Bet + ctrl.CtrlBet
							pl.Bank -= ctrl.CtrlBet
						} else {
							pl.IsFold = true
						}

						if !pl.IsAway {
							pl.TimeTurn += (now - g.Board.Deadline) + 10
							if pl.TimeTurn > 30 {
								pl.TimeTurn = 30
							}
						} else {
							pl.TimeTurn = 10
						}
						pl.HasActed = true
						pl.StoreCache()
						g.BroadcastPlayer(pl, false)

						pl = g.PlayersRing.Next(1)

						if pl.Bet == g.MaxBet && pl.HasActed {
							vabanks := true
							for _, p := range g.Players {
								p.HasActed, p.IsFold = false, false
								if p.Bank > 0 {
									vabanks = false
								}
							}
							if vabanks { // if everyone went "all in" but the river has not yet arrived
								stages = postriver
								copy(g.Board.Cards, g.Board.HiddenCards)
								g.Board.StoreCache()
								g.BroadcastBoard(&g.Board)
							}
							switch stages {
							case flop: // preflop to flop
								g.Board.Cards = append(g.Board.Cards, g.Board.HiddenCards[0], g.Board.HiddenCards[1], g.Board.HiddenCards[2])
							case turn, river: // flop to turn, turn to river
								g.Board.Cards = append(g.Board.Cards, g.Board.HiddenCards[stages+2])
							case postriver: //postriver evaluation
								if ok := g.PostRiver(); ok {
									GAME_LOOP = false
								}
								stages = 0
								continue
							}
							stages++
						}
						dur := time.Duration(pl.TimeTurn)
						g.Board.Deadline = time.Now().Add(time.Second * dur).Unix()
						g.TurnTimer.Reset(time.Second * dur)
						g.Board.StoreCache()
						g.BroadcastBoard(&g.Board)

					}
				case <-g.Shutdown:
					GAME_LOOP = false
				}
			}
		case ctrl := <-g.PlayerCh:
			g.Impl.Validate(ctrl)
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
		if v.Bank > 0 {
			newPlayers = append(newPlayers, v)
		} else {
			elims = append(elims, v)
			v.Place = -1
		}
	}
	g.Impl.PlayerOut(elims, len(g.Players))
	g.Players = newPlayers
	g.Idx -= len(elims)
	if len(newPlayers) == 1 {
		g.Impl.PlayerOut(newPlayers, 1)
		return true
	}
	g.DealNewHand()
	return
}

func (g *Game) DealNewHand() {
	g.Deck.Shuffle()
	g.Board.HiddenCards = g.Board.HiddenCards[:0] // cards that haven't come to the table yet (also string)
	g.Board.cards = g.Board.cards[:0]             // current cards on table for evaluating hand
	g.Board.Cards = g.Board.Cards[:0]             // current cards on table for json sending

	for i, v := range g.Players {
		v.cards = poker.NewHand()
		v.Place = i
		v.IsFold = false
		v.HasActed = false
	}
	c := g.Deck.Draw(7)
	for i := range c {
		g.CardsBuffer = append(g.CardsBuffer, c[i].String())
	}
	g.Board.cards = make([]poker.Card, 0, 7)

	g.Board.DealerPlace = g.PlayersRing.NextDealer(g.Board.DealerPlace, 1)

	g.PlayersRing.Next(1).Bet = g.Board.Blind
	g.PlayersRing.Next(1).Bet = g.Board.Blind * 2

	pl := g.PlayersRing.Next(1)

	g.Board.Deadline = time.Now().Add(time.Second * time.Duration(pl.TimeTurn)).Unix()
	g.BroadcastCh <- Ctrl{Brd: &g.Board}
	//send all players to all players (broadcast)
	for _, pl := range g.Players {
		g.BroadcastCh <- Ctrl{Plr: pl}
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
			poker.RankString()
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
	share := g.Board.Bank / i
	for j := range i {
		pl := g.Players[topPlaces[j].place]
		pl.Bank += share
		g.BroadcastCh <- Ctrl{IsExposed: true, Plr: pl}
	}
	time.Sleep(time.Second * 5)
}
