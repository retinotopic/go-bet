package hub

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/chehsunliu/poker"
	"github.com/retinotopic/go-bet/internal/player"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

func NewLobby() *Lobby {
	l := &Lobby{Players: make([]player.PlayUnit, 0), Occupied: make(map[int]bool), PlayerCh: make(chan player.PlayUnit), StartGame: make(chan struct{}, 1)}
	return l
}

const (
	preflop = iota
	flop
	turn
	river
)

type top struct {
	place  int
	rating int32
}
type Lobby struct {
	Players    []player.PlayUnit
	Admin      player.PlayUnit
	Board      player.PlayUnit
	SmallBlind int
	MaxBet     int
	Occupied   map[int]bool
	TurnTicker *time.Ticker
	sync.Mutex
	Order           int
	AdminOnce       sync.Once
	PlayerCh        chan player.PlayUnit
	StartGame       chan struct{}
	PlayerBroadcast chan player.PlayUnit
	isRating        bool
}

func (l *Lobby) LobbyWork() {
	fmt.Println("im in")
	for {
		select {
		case x := <-l.PlayerCh: // broadcoasting one seat to everyone
			for _, v := range l.Players {
				v.Conn.WriteJSON(x)
			}
		case <-l.StartGame:
			l.Game()
		}
	}
}

func (l *Lobby) ConnHandle(plr *player.PlayUnit) {
	fmt.Println("im in2")
	l.AdminOnce.Do(func() {
		if l.isRating {
			go l.tickerTillGame()
		} else {
			l.Admin = *plr
			plr.Admin = true
		}
	})

	defer func() {

	}()
	for _, v := range l.Players { // load current state of the game
		if v.Place != plr.Place {
			v = v.PrivateSend()
		}
		err := plr.Conn.WriteJSON(v)
		if err != nil {
			fmt.Println(err, "WriteJSON start")
		}
		fmt.Println("start")
	}
	for {
		ctrl := player.PlayUnit{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			fmt.Println(err, "conn read error")
			plr.Conn = nil
			break
		}
		if ctrl.Bet <= plr.Bankroll && ctrl.Bet >= 0 {
			plr.Bet = plr.Bet + ctrl.Bet
			l.PlayerCh <- *plr
		}
	}
}
func (l *Lobby) tickerTillGame() {
	ticker := time.NewTicker(time.Second * 1)
	i := 0
	for range ticker.C {
		if i >= 15 {
			l.StartGame <- struct{}{}
			return
		}
		i++
	}
}
func (l *Lobby) tickerTillNextTurn() {
	l.TurnTicker = time.NewTicker(time.Second * 1)
	timevalue := 0
	for range l.TurnTicker.C {
		timevalue++
		l.PlayerBroadcast <- l.Players[l.Order].SendTimeValue(timevalue)
		if timevalue >= 30 {
			l.TurnTicker.Stop()
			l.PlayerCh <- l.Players[l.Order]
		}
	}
}
func (l *Lobby) Game() {
	var stages int
	deck := poker.NewDeck()
	deck.Shuffle()
	randfuncs.NewSource().Shuffle(len(l.Players), func(i, j int) { l.Players[i], l.Players[j] = l.Players[j], l.Players[i] })

	for i, v := range l.Players {
		v.Cards = deck.Draw(2)
		v.Place = i
		if i+1 == len(l.Players) {
			v.NextPlayer = &l.Players[0]
		} else {
			v.NextPlayer = &l.Players[i+1]
		}
	}
	l.Board = player.PlayUnit{}
	l.Board.Cards = make([]poker.Card, 0, 7)
	l.PlayerBroadcast = make(chan player.PlayUnit)
	l.Players[0].Bet = l.SmallBlind
	l.Players[1].Bet = l.SmallBlind * 2
	l.Order = l.Players[2].Place

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
			if pl.Bet == l.MaxBet && pl.HasActed {
				switch turn {
				case preflop: // preflop to flop
					flopcard := deck.Draw(3)
					l.Board.Cards = append(l.Board.Cards, flopcard...)
				case flop, turn: // flop to turn, turn to river
					flopcard := deck.Draw(1)
					l.Board.Cards = append(l.Board.Cards, flopcard[0])
				case river:
					var emptyCard poker.Card
					topPlaces := make([]top, 0, 7)
					l.Board.Cards = append(l.Board.Cards, emptyCard, emptyCard)
					for _, v := range l.Players {
						if !pl.IsFold {
							l.Board.Cards[5] = v.Cards[0]
							l.Board.Cards[6] = v.Cards[1]
							eval := poker.Evaluate(l.Board.Cards)
							topPlaces = append(topPlaces, top{rating: eval, place: v.Place})
						}
					}
					sort.Slice(topPlaces, func(i, j int) bool {
						return topPlaces[i].rating < topPlaces[j].rating
					})
				}
			}
			l.Order = pl.Place
			stages++
			go l.tickerTillNextTurn()
		}
	}
}

// player broadcast method
func (l *Lobby) Broadcast() {
	for pb := range l.PlayerBroadcast {
		for _, v := range l.Players {
			if pb.Place != v.Place {
				pb = v.PrivateSend()
			}
			v.Conn.WriteJSON(pb)
			fmt.Println(v, "checkerv1")
		}
	}
}
