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

func (l *Lobby) tillTurnOrBlind() {
	for {
		select {
		case <-l.TurnTimer.C: // player turn timer
			l.pl.IsAway = true
			l.PlayerCh <- Ctrl{Plr: l.pl}
		case <-l.BlindTimer.C: // timer for blind raise
			l.BlindRaise = true
		case <-l.stop:
			return
		}
	}
}

func (l *Lobby) Game() {
	stages := 0
	var now int64
	var dur time.Duration

	var buf [32]byte
	bufs := make([]byte, 32)
	crand.Read(bufs) // generating cryptographically random slice of bytes
	copy(buf[:], bufs)
	rand := mrand.New(mrand.NewChaCha8(buf)) //  and using it as a ChaCha8 seed
	l.Deck = NewDeck(rand)                   // and using it as a source for mrand

	l.topPlaces = make([]top, 0, 8)    //---\
	l.winners = make([]PlayUnit, 0, 8) //	  >----- reusable memory for post-river evaluation
	l.losers = make([]PlayUnit, 0, 8)  //---/

	l.Board.HiddenCards = make([]string, 0, 5) // hidden cards that haven't come to the table yet (string)
	l.Board.cards = make([]poker.Card, 0, 7)   // poker.CardList for evaluating hand
	l.Board.Cards = make([]string, 0, 5)       // current cards on table for json sending
	for {
		select {
		case <-l.StartGameCh:

			l.Board.Blind = l.Board.Bank * int(stackShare[l.Blindlvl]) // initial blind
			l.InitialPlayerBank = l.Board.Bank

			l.stop = make(chan bool, 1)
			if l.Seats != [8]Seats{} { // if the game is in ranked mode, shuffle seats for randomness
				rand.Shuffle(8, func(i, j int) {
					l.Seats[i], l.Seats[j] = l.Seats[j], l.Seats[i]
				})
				for i, v := range l.Players { // give the player his starting stack
					v.Bank = l.Board.Bank
					v.Place = int(l.Seats[i].place)
				}
			}
			l.Board.Bank = 0
			l.DealNewHand()
			l.TurnTimer = time.NewTimer(time.Second * 10)
			l.BlindTimer = time.NewTimer(time.Duration(l.Board.Deadline) * time.Minute)
			go l.tillTurnOrBlind()
			defer func() { // prevent goroutine leakage
				l.stop <- true
			}()
			for GAME_LOOP := true; GAME_LOOP; { // MAIN GAME LOOP START
				select {
				case ctrl, ok := <-l.PlayerCh:
					if ok && ctrl.Plr == l.pl {

						l.TurnTimer.Stop()
						now = time.Now().Unix()
						l.pl = ctrl.Plr
						if ctrl.Ctrl > l.pl.Bank || ctrl.Ctrl < 0 { // if a player tries to cheat the system or just wants to quit, then we zero his bank
							l.pl.Bank = 0
							l.pl.Bet = 0
							ctrl.Ctrl = 0
						}

						if l.pl.Bet+ctrl.Ctrl >= l.Board.MaxBet {
							l.Board.MaxBet = l.pl.Bet
							l.pl.IsAway = false
							l.pl.Bet = l.pl.Bet + ctrl.Ctrl
							l.pl.Bank -= ctrl.Ctrl
						} else if ctrl.Ctrl > 0 {
							l.pl.IsAway = false
							l.pl.IsAllIn = true
							l.pl.Bet = l.pl.Bank
							l.pl.Bank = 0
						} else {
							l.pl.IsFold = true
						}

						if !l.pl.IsAway {
							l.pl.TimeTurn += (now - l.Board.Deadline) + 10
							if l.pl.TimeTurn > 30 {
								l.pl.TimeTurn = 30
							}
						} else {
							l.pl.TimeTurn = 10
						}
						l.pl.HasActed = true
						l.BroadcastPlayer(l.pl, false)

						l.pl = l.Next(1)

						//next poker stage if that player is the last one to raise the bet, and he acted at least once
						if l.pl.Bet == l.Board.MaxBet && l.pl.HasActed {
							AllIn := true
							for _, p := range l.Players {
								p.HasActed, p.IsFold = false, false
								l.Board.Bank += p.Bet // adding players bets to bank
								if p.Bank > 0 {
									AllIn = false
								}
							}
							if AllIn { // if everyone goes "all in" and river has not yet arrived
								stages = postriver
								copy(l.Board.Cards, l.Board.HiddenCards)
								l.Board.Deadline = 0
								l.BroadcastBoard(&l.Board)
							}
							switch stages {
							case flop: // preflop to flop
								l.Board.Cards = append(l.Board.Cards, l.Board.HiddenCards[:3]...)
							case turn, river: // flop to turn, turn to river
								l.Board.Cards = append(l.Board.Cards, l.Board.HiddenCards[stages+2])
							case postriver: // postriver evaluation
								if ok := l.PostRiver(); ok { // if postriver() returns true, then end the current game
									GAME_LOOP = false
									continue
								}
								stages = -1
							}
							stages++
						}
						dur = time.Duration(l.pl.TimeTurn)                          //------------\
						l.Board.Deadline = time.Now().Add(time.Second * dur).Unix() //			   \
						l.TurnTimer.Reset(time.Second * dur)                        //			   /----- notifying the player of the start of his turn
						l.BroadcastBoard(&l.Board)                                  //------------/
					}
				case <-l.Shutdown:
					return
				}
			} // MAIN GAME LOOP END
		case ctrl := <-l.PlayerCh: // pre-game board tuning
			l.Validate(ctrl)
		case <-l.Shutdown:
			return
		}
	}

}
func (l *Lobby) PostRiver() (gameOver bool) {
	l.CalcWinners()
	l.winners = l.winners[:0]
	l.losers = l.losers[:0]
	for i := range l.Players {
		if l.Players[i].Bank > 0 {
			l.winners = append(l.winners, l.Players[i])
		} else {
			l.losers = append(l.losers, l.Players[i])
		}
	}
	go l.PlayerOut(l.losers, len(l.Players))
	l.Players = l.Players[:(len(l.winners))]
	copy(l.Players, l.winners)
	l.Idx -= len(l.losers)
	if len(l.winners) == 1 {
		l.PlayerOut(l.winners, 1)
		return true
	}
	l.DealNewHand()
	return
}

func (l *Lobby) DealNewHand() {
	l.Deck.Shuffle()

	l.Board.Cards = l.Board.Cards[:0]

	l.Deck.Draw(l.Board.cards[2:], l.Board.HiddenCards) // fill hidden cards
	for _, v := range l.Players {
		l.Deck.Draw(v.CardsEval, v.Cards[1:]) // fill players cards
		v.IsFold = false
		v.HasActed = false
		v.IsAllIn = false
	}
	l.Board.DealerPlace = l.NextDealer(l.Board.DealerPlace, 1) // evaluate next dealer place

	if l.BlindRaise { // blind raise
		l.Blindlvl++
		if len(stackShare) > l.Blindlvl {
			l.Board.Blind = l.InitialPlayerBank * int(stackShare[l.Blindlvl])
			l.BlindTimer.Reset(time.Minute * 5)
		} else {
			l.BlindTimer.Stop()
		}
		l.BlindRaise = false
	}

	l.Next(1).Bet = l.Board.Blind // small blind

	l.Next(1).Bet = l.Board.Blind * 2 // big blind

	l.pl = l.Next(1)

	//send all current players to all users in room
	for i, _ := range l.Players {
		go l.BroadcastPlayer(&l.Players[i], false)
	}
}

func (l *Lobby) CalcWinners() {
	l.topPlaces = l.topPlaces[:0]
	for i, v := range l.Players { // evaluate players cards and rank them
		if !v.IsFold && !v.IsAway {
			l.Board.cards[0] = v.CardsEval[0]
			l.Board.cards[1] = v.CardsEval[1]

			eval := l.Board.cards.Evaluate()
			v.Cards[0] = poker.GetHandRank(eval).String()
			l.topPlaces = append(l.topPlaces, top{rating: eval, place: i})
		}
		v.IsFold = false
	}
	sort.Slice(l.topPlaces, func(i, j int) bool {
		return l.topPlaces[i].rating < l.topPlaces[j].rating
	})
	j := 0
	for ; j == 0 || l.topPlaces[j].rating == l.topPlaces[j-1].rating; j++ {
	}
	share := l.Board.Bank / j

	for i := range j { // distribute the pot between winners
		pl := l.Players[l.topPlaces[i].place]
		pl.Bank += share
		go l.BroadcastPlayer(pl, true)
	}
	l.Board.Bank = 0 // reset the board bank
	time.Sleep(time.Second * 5)
}
