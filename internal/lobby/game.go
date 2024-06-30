package lobby

import (
	"math"
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
		l.PlayerBroadcast <- l.Players[l.PlayersRing.Idx].SendTimeValue(timevalue)
		if timevalue >= 30 {
			l.TurnTicker.Stop()
			l.PlayerCh <- *l.Players[l.PlayersRing.Idx]
		}
	}
}
func (l *Lobby) Game() {
	var stages int
	randfuncs.NewSource().Shuffle(l.LenPlayers, func(i, j int) {
		l.Players[i], l.Players[j] = l.Players[j], l.Players[i]
	})
	l.PlayerBroadcast = make(chan PlayUnit)
	l.DealNewHand()
	go l.tickerTillNextTurn()
	for pl := range l.PlayerCh { // evaluating hand
		if pl.Place == l.PlayersRing.Idx {
			l.TurnTicker.Stop()
			if pl.Bet >= l.MaxBet {
				l.MaxBet = pl.Bet
			} else {
				pl.IsFold = true
			}
			pl.HasActed = true
			for pl.IsFold {
				pl = *l.PlayersRing.Next(1)
			}
			if pl.Bet == l.MaxBet && pl.HasActed {
				switch turn {
				case flop: // preflop to flop
					flopcard := l.Deck.Draw(3)
					l.Board.Cards = append(l.Board.Cards, flopcard...)
				case turn, river: // flop to turn, turn to river
					flopcard := l.Deck.Draw(1)
					l.Board.Cards = append(l.Board.Cards, flopcard[0])
				case postriver:
					l.PostRiver()
					stages = -1
				}
				for _, v := range l.Players {
					v.HasActed = false
				}
			}
			stages++
			go l.tickerTillNextTurn()
		}
	}
}
func (l *Lobby) PostRiver() {
	l.CalcWinners()
	newPlayers := make([]*PlayUnit, 0, 8)
	elims := make([]*PlayUnit, 0, 8)
	var last int
	for i, v := range l.Players {
		if v.Bankroll > 0 {
			newPlayers = append(newPlayers, v)
		} else {
			last = len(l.Players) - i
			elims = append(elims, v)
		}
	}
	if len(elims) != 0 {
		l.CalcRating(elims, last)
	}
	if len(newPlayers) == 1 {
		l.CalcRating(newPlayers, 1)
		return
	}
	l.Players = newPlayers
	l.DealNewHand()
}

func (l *Lobby) DealNewHand() {
	l.Deck = poker.NewDeck()
	l.Deck.Shuffle()

	for i, v := range l.Players {
		v.Cards = l.Deck.Draw(2)
		v.Place = i
	}
	l.Board = PlayUnit{}
	l.Board.Cards = make([]poker.Card, 0, 7)

	l.PlayersRing.Next(1).Bet = l.SmallBlind
	l.PlayersRing.Next(2).Bet = l.SmallBlind * 2
	//send all players to all players (broadcast)
	for _, pl := range l.Players {
		l.PlayerBroadcast <- *pl
	}
	l.TurnTicker.Reset(time.Second * 30)
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
func (l *Lobby) CalcRating(plr []*PlayUnit, place int) {
	baseChange := 30
	middlePlace := float64(l.LenPlayers+1) / 2
	_ = int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
	//l.Players[2].User_id
}
