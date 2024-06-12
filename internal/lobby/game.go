package lobby

import (
	"sort"
	"time"

	"github.com/chehsunliu/poker"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

func (l *Lobby) tickerTillNextTurn() {
	l.TurnTicker = time.NewTicker(time.Second * 1)
	timevalue := 0
	for range l.TurnTicker.C {
		timevalue++
		l.PlayerBroadcast <- l.Players[l.Order].SendTimeValue(timevalue)
		if timevalue >= 30 {
			l.TurnTicker.Stop()
			l.PlayerCh <- *l.Players[l.Order]
		}
	}
}
func (l *Lobby) Game() {
	var stages int
	randfuncs.NewSource().Shuffle(len(l.Players), func(i, j int) { l.Players[i], l.Players[j] = l.Players[j], l.Players[i] })
	l.PlayerBroadcast = make(chan PlayUnit)
	l.DealNewHand()
	go l.tickerTillNextTurn()
	for pl := range l.PlayerCh { // evaluating hand
		if pl.Place == l.Order {
			l.TurnTicker.Stop()
			if pl.Bet >= l.MaxBet {
				l.MaxBet = pl.Bet
			} else {
				pl.IsFold = true
			}
			pl.HasActed = true
			for pl.IsFold {
				pl = *pl.NextPlayer
			}
			l.Order = pl.Place
			if pl.Bet == l.MaxBet && pl.HasActed {
				switch turn {
				case flop: // preflop to flop
					flopcard := l.Deck.Draw(3)
					l.Board.Cards = append(l.Board.Cards, flopcard...)
				case turn, river: // flop to turn, turn to river
					flopcard := l.Deck.Draw(1)
					l.Board.Cards = append(l.Board.Cards, flopcard[0])
				case postriver:
					l.CalcWinners()
					newPlayers := make([]*PlayUnit, 0, 8)
					ch := make(chan bool, 1)
					stages = -1
					for i, v := range l.Players {
						go func() {
							if v.Bankroll > 0 {
								newPlayers = append(newPlayers, v)
							}
							if len(l.Players)-1 == i {
								ch <- true
							}
						}()
						l.PlayerBroadcast <- *v ////////////// calculate rating
					}
					<-ch
					if len(newPlayers) == 1 {
						/////////////////////////////////////
					}
					l.Players = newPlayers
					l.DealNewHand()
				}
			}
			stages++
			go l.tickerTillNextTurn()
		}
	}
}
func (l *Lobby) DealNewHand() {
	l.Deck = poker.NewDeck()
	l.Deck.Shuffle()

	for i, v := range l.Players {
		v.Cards = l.Deck.Draw(2)
		v.Place = i
		if i+1 == len(l.Players) {
			v.NextPlayer = l.Players[0]
		} else {
			v.NextPlayer = l.Players[i+1]
		}
	}
	l.Board = PlayUnit{}
	l.Board.Cards = make([]poker.Card, 0, 7)

	l.Players[0].Bet = l.SmallBlind
	l.Players[0].NextPlayer.Bet = l.SmallBlind * 2
	l.Order = l.Players[0].NextPlayer.NextPlayer.Place
	//send all players to all players (broadcast)
}

func (l *Lobby) CalcWinners() {
	var emptyCard poker.Card
	topPlaces := make([]top, 0, 7)
	l.Board.Cards = append(l.Board.Cards, emptyCard, emptyCard)
	for i, v := range l.Players {
		if !v.IsFold {
			l.Board.Cards[5] = v.Cards[0]
			l.Board.Cards[6] = v.Cards[1]
			eval := poker.Evaluate(l.Board.Cards)
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
	share := l.Board.Bankroll / i
	for pl := range i {
		l.Players[topPlaces[pl].place].Bankroll += share
	}
}
