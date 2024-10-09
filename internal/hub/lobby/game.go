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
	rating uint16
}
type Implementor interface {
	Validate(Ctrl)
	PlayerOut([]*PlayUnit, int)
}
type Seats struct {
	place      uint8 // zero to seven
	isOccupied bool
}
type Game struct {
	*Lobby
	Seats       [8]Seats
	Deck        *Deck
	pl          *PlayUnit   //current player
	MaxBet      int         //
	TurnTimer   *time.Timer //
	BlindTimer  *time.Timer //
	StartGameCh chan bool
	stop        chan bool
	BlindRaise  bool
	Impl        Implementor
}

func (g *Game) tillTurnOrBlind() {
	for {
		select {
		case <-g.TurnTimer.C:
			g.pl.IsAway = true
			g.PlayerCh <- Ctrl{Plr: g.pl}
		case <-g.BlindTimer.C:
			g.BlindRaise = true
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
			var now int64
			var dur time.Duration

			g.Board.DealerPlace = 0
			g.Board.Blind = g.Board.Bank * int(stackShare[g.Blindlvl]) // initial blind

			g.Board.HiddenCards = make([]string, 5) // hidden cards that haven't come to the table yet (string)
			g.Board.cards = make([]poker.Card, 7)   // poker.CardList for evaluating hand
			g.Board.Cards = make([]string, 0, 5)    // current cards on table for json sending

			g.stop = make(chan bool, 1)
			randsrc := rand.NewSource(int64(new(maphash.Hash).Sum64()))
			rand.New(randsrc).Shuffle(8, func(i, j int) { // shuffle player seats
				g.Seats[i], g.Seats[j] = g.Seats[j], g.Seats[i]
			})
			for i, v := range g.Players { // give the player his starting stack
				v.Bank = g.Board.Bank
				v.Place = int(g.Seats[i].place)
			}
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
					if ok && ctrl.Plr == g.pl {

						g.TurnTimer.Stop()
						now = time.Now().Unix()
						g.pl = ctrl.Plr
						if ctrl.CtrlInt > g.pl.Bank || ctrl.CtrlInt < 0 { // if a player tries to cheat the system or just wants to quit, then we zero his bank
							g.pl.Bank = 0
							g.pl.Bet = 0
							ctrl.CtrlInt = 0
						}

						if g.pl.Bet+ctrl.CtrlInt >= g.MaxBet {
							g.MaxBet = g.pl.Bet
							g.pl.IsAway = false
							g.pl.Bet = g.pl.Bet + ctrl.CtrlInt
							g.pl.Bank -= ctrl.CtrlInt
						} else {
							g.pl.IsFold = true
						}

						if !g.pl.IsAway {
							g.pl.TimeTurn += (now - g.Board.Deadline) + 10
							if g.pl.TimeTurn > 30 {
								g.pl.TimeTurn = 30
							}
						} else {
							g.pl.TimeTurn = 10
						}
						g.pl.HasActed = true
						g.pl.StoreCache()
						g.BroadcastPlayer(g.pl, false)

						g.pl = g.PlayersRing.Next(1)

						if g.pl.Bet == g.MaxBet && g.pl.HasActed {
							vabanks := true
							for _, p := range g.Players {
								p.HasActed, p.IsFold = false, false
								if p.Bank > 0 {
									vabanks = false
								}
							}
							if vabanks { // if everyone goes "all in" and river has not yet arrived
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
									continue
								}
								stages = -1
							}
							stages++
						}
						dur = time.Duration(g.pl.TimeTurn)
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
		}
	}
	g.Impl.PlayerOut(elims, len(g.Players))
	g.itermtx.Lock()
	g.Players = g.Players[:(len(newPlayers))]
	copy(g.Players, newPlayers)
	g.itermtx.Unlock()
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
	g.Board.Cards = g.Board.Cards[:0] // current cards on table for json sending
	g.Deck.Draw(g.Board.cards[2:], g.Board.HiddenCards)
	for _, v := range g.Players {
		g.Deck.Draw(v.cards, v.Cards[1:])
		v.IsFold = false
		v.HasActed = false
	}

	g.Board.DealerPlace = g.PlayersRing.NextDealer(g.Board.DealerPlace, 1)
	if g.BlindRaise {
		g.Blindlvl++
		if len(stackShare) > g.Blindlvl {
			g.Board.Blind *= int(stackShare[g.Blindlvl])
			g.BlindTimer.Reset(time.Minute * 5)
		} else {
			g.BlindTimer.Stop()
		}
		g.BlindRaise = false
	}
	g.PlayersRing.Next(1).Bet = g.Board.Blind     // small blind
	g.PlayersRing.Next(1).Bet = g.Board.Blind * 2 // big blind

	g.pl = g.PlayersRing.Next(1)
	//send all players to all players (broadcast)
	for _, pl := range g.Players {
		pl.StoreCache()
		g.BroadcastPlayer(pl, false)
	}
}

func (g *Game) CalcWinners() {
	topPlaces := make([]top, 0, 8)

	for i, v := range g.Players {
		if !v.IsFold && !v.IsAway {
			g.Board.cards[0] = v.cards[0]
			g.Board.cards[1] = v.cards[1]

			eval := g.Board.cards.Evaluate()
			v.Cards[0] = poker.GetHandRank(eval).String()
			topPlaces = append(topPlaces, top{rating: eval, place: i})
		}
		v.IsFold = false
	}
	sort.Slice(topPlaces, func(i, j int) bool {
		return topPlaces[i].rating < topPlaces[j].rating
	})
	j := 0
	for ; j == 0 || topPlaces[j].rating == topPlaces[j-1].rating; j++ {
	}
	share := g.Board.Bank / j
	for i := range j {
		pl := g.Players[topPlaces[i].place]
		pl.Bank += share
		g.pl.StoreCache()
		g.BroadcastPlayer(g.pl, true)
	}
	time.Sleep(time.Second * 5)
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
