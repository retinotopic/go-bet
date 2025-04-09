package lobby

import (
	mrand "math/rand/v2"
	"sort"
	"time"
)

type Top struct {
	Place  int
	Rating int
	Exiled bool
}

type Seats struct {
	place      uint8 // zero to seven
	isOccupied bool  // seat occupied or not
}

func (l *Lobby) tillTurnOrBlind() {
	for {
		select {
		case <-l.TurnTimer.C: // player turn timer
			l.AllUsers[l.Pl].IsAway = true
			l.PlayerCh <- Ctrl{Plr: l.Pl}
		case <-l.BlindTimer.C: // timer for blind raise
			l.BlindRaise = true
		case <-l.Stop:
			return
		}
	}
}

func (l *Lobby) Game() {
	stages := 0
	var now int64
	var dur time.Duration
	HiddenCards := l.Board.CardsEval[2:]
	for {
		select {
		case <-l.StartGameCh:
			l.Board.Blind = l.Board.Bank * int(stackShare[l.Blindlvl]) // initial blind
			l.InitialPlayerBank = l.Board.Bank

			if l.Seats != [8]Seats{} { // if the game is in ranked mode, shuffle seats for randomness
				mrand.Shuffle(8, func(i, j int) {
					l.Seats[i], l.Seats[j] = l.Seats[j], l.Seats[i]
				})
				for i, idx := range l.Players { // give the player his starting stack
					v := &l.AllUsers[idx]
					v.Bank = l.Board.Bank
					v.Place = int(l.Seats[i].place)
				}
			}
			l.Board.Bank = 0
			l.DealNewHand()

			l.TurnTimer.Reset(time.Second * 10)
			l.BlindTimer.Reset(time.Duration(l.Board.Deadline) * time.Minute)

			go l.tillTurnOrBlind()
			defer func() {
				l.Stop <- true
			}()
			for GAME_LOOP := true; GAME_LOOP; { // MAIN GAME LOOP START
				select {
				case ctrl, ok := <-l.PlayerCh:
					if ok && ctrl.Plr == l.Pl {
						pl := &l.AllUsers[l.Pl]

						l.TurnTimer.Stop()
						now = time.Now().Unix()
						if ctrl.Ctrl > pl.Bank || ctrl.Ctrl < 0 { // if a player tries to cheat the system or just wants to quit, then we zero his bank
							pl.Bank = 0
							pl.Bet = 0
							ctrl.Ctrl = 0
						}

						if pl.Bet+ctrl.Ctrl >= l.Board.MaxBet {
							l.Board.MaxBet = pl.Bet
							pl.IsAway = false
							pl.Bet = pl.Bet + ctrl.Ctrl
							pl.Bank -= ctrl.Ctrl
						} else if ctrl.Ctrl > 0 {
							pl.IsAway = false
							pl.IsAllIn = true
							pl.Bet = pl.Bank
							pl.Bank = 0
						} else {
							pl.IsFold = true
						}

						if !pl.IsAway {
							pl.TimeTurn += (now - l.Board.Deadline) + 10
							if pl.TimeTurn > 30 {
								pl.TimeTurn = 30
							}
						} else {
							pl.TimeTurn = 10
						}
						pl.HasActed = true
						l.BroadcastPlayer(l.Pl, false)

						l.Pl = l.Next(1)
						pl = &l.AllUsers[l.Pl]

						//next poker stage if that player is the last one to raise the bet, and he acted at least once
						if pl.Bet == l.Board.MaxBet && pl.HasActed {
							AllIn := true
							for _, idx := range l.Players {
								p := &l.AllUsers[idx]
								p.HasActed, p.IsFold = false, false
								l.Board.Bank += p.Bet // adding players bets to bank
								if p.Bank > 0 {
									AllIn = false
								}
							}
							if AllIn { // if everyone goes "all in" and river has not yet arrived
								stages = postriver
								copy(l.Board.Cards, l.Board.CardsEval[2:])
								l.Board.Deadline = 0
								l.BroadcastBoard()
							}
							switch stages {
							case flop: // preflop to flop
								l.Board.Cards = append(l.Board.Cards, HiddenCards[:3]...)
							case turn, river: // flop to turn, turn to river
								l.Board.Cards = append(l.Board.Cards, HiddenCards[stages+2])
							case postriver: // postriver evaluation
								if ok := l.PostRiver(); ok { // if postriver() returns true, then end the current game
									GAME_LOOP = false
									continue
								}
								stages = -1
							}
							stages++
						}
						dur = time.Duration(pl.TimeTurn)                            //------------\
						l.Board.Deadline = time.Now().Add(time.Second * dur).Unix() //			   \
						l.TurnTimer.Reset(time.Second * dur)                        //			   /----- notifying the player of the start of his turn
						l.BroadcastBoard()                                          //------------/
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
	l.Winners = l.Winners[:0]
	l.Losers = l.Losers[:0]
	for _, idx := range l.Players {
		p := &l.AllUsers[idx]
		if p.Bank > 0 {
			l.Winners = append(l.Winners, idx)
		} else {
			l.Losers = append(l.Losers, idx)
		}
	}
	go l.PlayerOut(l.Losers, len(l.Players))
	l.Players = l.Players[:(len(l.Winners))]
	copy(l.Players, l.Winners)
	l.Idx -= len(l.Losers)
	if len(l.Winners) == 1 {
		l.PlayerOut(l.Winners, 1)
		return true
	}
	l.DealNewHand()
	return
}

func (l *Lobby) DealNewHand() {
	l.Deck.Shuffle()

	l.Board.Cards = l.Board.Cards[:0]

	l.Deck.Draw(l.Board.CardsEval) // fill hidden cards
	for _, idx := range l.Players {
		v := &l.AllUsers[idx]
		l.Deck.Draw(v.Cards[2:]) // fill players cards
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

	l.AllUsers[l.Next(1)].Bet = l.Board.Blind // small blind

	l.AllUsers[l.Next(1)].Bet = l.Board.Blind * 2 // big blind

	l.Pl = l.Next(1)

	//send all current players to all users in room
	for _, v := range l.Players {
		go l.BroadcastPlayer(v, false)
	}
}

func (l *Lobby) CalcWinners() {
	l.TopPlaces = l.TopPlaces[:0]
	maxv1, maxv2 := 0, 0
	for i, idx := range l.Players { // evaluate players cards and rank them
		v := &l.AllUsers[idx]
		if !v.IsFold && !v.IsAway {
			if i == 0 {
				maxv1, maxv2 = idx, idx
			}
			l.Board.CardsEval[0] = v.Cards[0]
			l.Board.CardsEval[1] = v.Cards[1]

			eval := l.Board.CardsEval.Evaluate()
			l.TopPlaces = append(l.TopPlaces, Top{Rating: int(eval), Place: idx})
			if maxv1 < v.Bet {
				maxv2 = maxv1
				maxv1 = v.Bet
			} else if maxv2 < v.Bet && v.Bet < maxv1 {
				maxv2 = v.Bet
			}
		}
		v.IsFold = false
	}
	sort.Slice(l.TopPlaces, func(i, j int) bool {
		return l.TopPlaces[i].Rating < l.TopPlaces[j].Rating
	})
	l.CalcSidePots(maxv1, maxv2)

	for i := range l.TopPlaces {
		go l.BroadcastPlayer(l.TopPlaces[i].Place, true)
	}
	l.Board.Bank = 0 // reset the board bank
	time.Sleep(time.Second * 5)
}
func (l *Lobby) CalcSidePots(maxv1 int, maxv2 int) {
	dif := maxv1 - maxv2
	maxl := maxv1
	i, j, share, shcount, maxv1, maxv2, lastRating := 0, 0, 0, 0, 0, 0, -1
	for {
		v := &l.AllUsers[l.TopPlaces[i].Place]
		if v.Bet == maxl {
			if !l.TopPlaces[i].Exiled {
				if lastRating == -1 {
					lastRating = l.TopPlaces[i].Rating
				}
				if l.TopPlaces[i].Rating < lastRating {
					l.TopPlaces[i].Exiled = true
				} else {
					shcount++
				}
			}
			v.SidePots[j] = dif
			share += dif
			v.Bet -= dif
		}
		if maxv1 < v.Bet {
			maxv2 = maxv1
			maxv1 = v.Bet
		} else if maxv2 < v.Bet && v.Bet < maxv1 {
			maxv2 = v.Bet
		}
		if i == len(l.TopPlaces)-1 {
			share = share / shcount
			done := true
			for k := range l.TopPlaces {
				v := &l.AllUsers[l.TopPlaces[k].Place]
				if !l.TopPlaces[k].Exiled && v.SidePots[j] != 0 {
					v.Bank += share
				}
				if v.Bet != 0 {
					done = false
				}
				v.SidePots[j] = 0
			}
			if done {
				break
			}
			dif = maxv1 - maxv2
			maxl = maxv1
			maxv1, maxv2, share, shcount, i, lastRating = 0, 0, 0, 0, -1, -1
			j++
		}
		i++
	}
}
