package lobby

import (
	crand "crypto/rand"
	mrand "math/rand/v2"
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
	isOccupied bool  // seat occupied or not
}
type Game struct {
	*Lobby
	InitialPlayerBank int
	Seats             [8]Seats
	topPlaces         []top
	winners           []*PlayUnit
	losers            []*PlayUnit
	Deck              *Deck
	pl                *PlayUnit //current player
	MaxBet            int
	TurnTimer         *time.Timer
	BlindTimer        *time.Timer
	StartGameCh       chan bool
	stop              chan bool
	BlindRaise        bool
	Impl              Implementor
}

func (g *Game) tillTurnOrBlind() {
	for {
		select {
		case <-g.TurnTimer.C: // player turn timer
			g.pl.IsAway = true
			g.PlayerCh <- Ctrl{Plr: g.pl}
		case <-g.BlindTimer.C: // timer for blind raise
			g.BlindRaise = true
		case <-g.stop:
			return
		}
	}
}

func (g *Game) Game() {
	stages := 0
	var now int64
	var dur time.Duration

	var buf [32]byte
	bufs := make([]byte, 32)
	crand.Read(bufs)
	copy(buf[:], bufs)
	rand := mrand.New(mrand.NewChaCha8(buf))
	g.Deck = NewDeck(rand)

	g.topPlaces = make([]top, 0, 8)     //---\
	g.winners = make([]*PlayUnit, 0, 8) //	  >----- reusable memory for post-river evaluation
	g.losers = make([]*PlayUnit, 0, 8)  //---/

	g.Board.HiddenCards = make([]string, 0, 5) // hidden cards that haven't come to the table yet (string)
	g.Board.cards = make([]poker.Card, 0, 7)   // poker.CardList for evaluating hand
	g.Board.Cards = make([]string, 0, 5)       // current cards on table for json sending
	for {
		select {
		case <-g.StartGameCh:

			g.Board.Blind = g.Board.Bank * int(stackShare[g.Blindlvl]) // initial blind
			g.InitialPlayerBank = g.Board.Bank
			g.Board.Bank = 0

			g.stop = make(chan bool, 1)
			if g.Seats != [8]Seats{} { // if the game is in ranked mode, shuffle seats for randomness
				rand.Shuffle(8, func(i, j int) {
					g.Seats[i], g.Seats[j] = g.Seats[j], g.Seats[i]
				})
				for i, v := range g.Players { // give the player his starting stack
					v.Bank = g.Board.Bank
					v.Place = int(g.Seats[i].place)
				}
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
						g.BroadcastPlayer(g.pl, false)

						g.pl = g.PlayersRing.Next(1)

						//next poker stage if that player is the last one to raise the bet, and he acted at least once
						if g.pl.Bet == g.MaxBet && g.pl.HasActed {
							vabanks := true
							for _, p := range g.Players {
								p.HasActed, p.IsFold = false, false
								g.Board.Bank += p.Bet // adding players bets to bank
								if p.Bank > 0 {
									vabanks = false
								}
							}
							if vabanks { // if everyone goes "all in" and river has not yet arrived
								stages = postriver
								copy(g.Board.Cards, g.Board.HiddenCards)
								g.BroadcastBoard(&g.Board)
							}
							switch stages {
							case flop: // preflop to flop
								g.Board.Cards = append(g.Board.Cards, g.Board.HiddenCards[0], g.Board.HiddenCards[1], g.Board.HiddenCards[2])
							case turn, river: // flop to turn, turn to river
								g.Board.Cards = append(g.Board.Cards, g.Board.HiddenCards[stages+2])
							case postriver: // postriver evaluation
								if ok := g.PostRiver(); ok { // if postriver() returns true, then end the current game
									GAME_LOOP = false
									continue
								}
								stages = -1
							}
							stages++
						}
						dur = time.Duration(g.pl.TimeTurn)                          //------------\
						g.Board.Deadline = time.Now().Add(time.Second * dur).Unix() //			   \
						g.TurnTimer.Reset(time.Second * dur)                        //			   /----- notifying the player of the start of his turn
						g.BroadcastBoard(&g.Board)                                  //------------/
					}
				case <-g.Shutdown:
					return
				}
			}
		case ctrl := <-g.PlayerCh: // pre-game board tuning
			g.Impl.Validate(ctrl)
		case <-g.Shutdown:
			return
		}
	}

}
func (g *Game) PostRiver() (gameOver bool) {
	g.CalcWinners()
	g.winners = g.winners[:0]
	g.losers = g.losers[:0]
	for _, v := range g.Players {
		if v.Bank > 0 {
			g.winners = append(g.winners, v)
		} else {
			g.losers = append(g.losers, v)
		}
	}
	go g.Impl.PlayerOut(g.losers, len(g.Players))
	g.Players = g.Players[:(len(g.winners))]
	copy(g.Players, g.winners)
	g.Idx -= len(g.losers)
	if len(g.winners) == 1 {
		g.Impl.PlayerOut(g.winners, 1)
		return true
	}
	g.DealNewHand()
	return
}

func (g *Game) DealNewHand() {
	g.Deck.Shuffle()

	g.Board.Cards = g.Board.Cards[:0]
	g.Deck.Draw(g.Board.cards[2:], g.Board.HiddenCards)
	for _, v := range g.Players {
		g.Deck.Draw(v.CardsEval, v.Cards[1:])
		v.IsFold = false
		v.HasActed = false
	}
	g.Board.DealerPlace = g.PlayersRing.NextDealer(g.Board.DealerPlace, 1)
	if g.BlindRaise { // blind raise
		g.Blindlvl++
		if len(stackShare) > g.Blindlvl {
			g.Board.Blind = g.InitialPlayerBank * int(stackShare[g.Blindlvl])
			g.BlindTimer.Reset(time.Minute * 5)
		} else {
			g.BlindTimer.Stop()
		}
		g.BlindRaise = false
	}

	g.PlayersRing.Next(1).Bet = g.Board.Blind     // small blind
	g.PlayersRing.Next(1).Bet = g.Board.Blind * 2 // big blind

	g.pl = g.PlayersRing.Next(1)
	//send all current players to all users in room
	for _, pl := range g.Players {
		g.BroadcastPlayer(pl, false)
	}
}

func (g *Game) CalcWinners() {
	g.topPlaces = g.topPlaces[:0]
	for i, v := range g.Players {
		if !v.IsFold && !v.IsAway {
			g.Board.cards[0] = v.CardsEval[0]
			g.Board.cards[1] = v.CardsEval[1]

			eval := g.Board.cards.Evaluate()
			v.Cards[0] = poker.GetHandRank(eval).String()
			g.topPlaces = append(g.topPlaces, top{rating: eval, place: i})
		}
		v.IsFold = false
	}
	sort.Slice(g.topPlaces, func(i, j int) bool {
		return g.topPlaces[i].rating < g.topPlaces[j].rating
	})
	j := 0
	for ; j == 0 || g.topPlaces[j].rating == g.topPlaces[j-1].rating; j++ {
	}
	share := g.Board.Bank / j
	for i := range j {
		pl := g.Players[g.topPlaces[i].place]
		pl.Bank += share
	}
	for i := range j {
		pl := g.Players[g.topPlaces[i].place]
		g.BroadcastPlayer(pl, true)
	}
	g.Board.Bank = 0
	time.Sleep(time.Second * 5)
}
