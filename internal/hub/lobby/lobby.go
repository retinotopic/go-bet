package lobby

import (
	"bytes"
	"context"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/coder/websocket"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

const (
	flop = iota
	turn
	river
	postriver
)

type Ctrl struct { //control union
	IsExposed bool      `json:"-"` // means whether the cards should be shown to everyone
	Place     int       `json:"Place"`
	CtrlBet   int       `json:"CtrlBet"`
	Message   string    `json:"Message"`
	Plr       *PlayUnit `json:"-"`
}

type Lobby struct {
	PlayersRing
	MapTable     *csmap.CsMap[string, *PlayUnit] // id to place mapping
	HasBegun     bool
	PlayerCh     chan Ctrl
	checkTimeout *time.Ticker
	lastResponse time.Time
	ValidateCh   chan Ctrl
	Shutdown     chan bool
	Url          uint64
}

func (l *Lobby) LobbyStart() {
	gm := &Game{Lobby: l}
	go gm.Game()
	l.checkTimeout = time.NewTicker(time.Minute * 3)
	l.Shutdown = make(chan bool, 3)
	l.PlayerCh = make(chan Ctrl)
	timeout := time.Minute * 4
	for {
		select {
		case <-l.checkTimeout.C:
			if time.Since(l.lastResponse) > timeout {
				for range 3 { //shutdowns,two for game and one for lobby
					l.Shutdown <- true
				}
			}
		case <-l.Shutdown:
			return
		}

	}
}

func (c *Lobby) HandleConn(plr *PlayUnit) {
	defer func() {
		c.MapTable.DeleteIf(plr.User_id, func(value *PlayUnit) bool { return value.Place == -1 }) //if the seat is unoccupied, delete from map
		plr.Conn.CloseNow()
	}()
	byteplr, err := json.Marshal(plr)
	if err != nil {
		return
	}
	bytebrd, err := json.Marshal(c.Board)
	if err != nil {
		return
	}
	go writeTimeout(time.Second*5, plr.Conn, bytebrd)
	go writeTimeout(time.Second*5, plr.Conn, byteplr)

	for _, v := range c.Players { // load current state of the game
		if v != plr {
			writeTimeout(time.Second*5, plr.Conn, EmptyCards(v.GetCache()))
		}
	}
	for { // listening for actions
		ctrl := Ctrl{}
		_, data, err := plr.Conn.Read(context.TODO())
		if err != nil {
			break
		}
		err = json.Unmarshal(data, &ctrl)
		if err != nil {
			break
		}
		ctrl.Plr = plr
		plr.IsAway = false
		c.PlayerCh <- ctrl
	}
}
func (l *Lobby) BroadcastPlayer(plr PlayUnit, isExposed bool) (err error) {
	b, err := json.Marshal(plr)
	if err != nil {
		return err
	}
	go writeTimeout(time.Second*5, plr.Conn, b)
	if !isExposed {
		EmptyCards(b)
	}
	l.m.Range(func(key string, val *PlayUnit) (stop bool) { // spectators
		go writeTimeout(time.Second*5, val.Conn, b)
		return
	})
	return
}

func (l *Lobby) BroadcastBoard(board *GameBoard) (err error) {
	b, err := json.Marshal(board)
	if err != nil {
		return err
	}
	l.m.Range(func(key string, val *PlayUnit) (stop bool) { // spectators
		go writeTimeout(time.Second*5, val.Conn, b)
		return
	})

	return
}
func writeTimeout(timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
func EmptyCards(data []byte) (data2 []byte) { // in order to not marshaling twice, but for the cards to be empty
	start := bytes.IndexByte(data, '[')
	if start == -1 {
		return data
	}
	start++
	end := bytes.IndexByte(data, ']')
	if end == -1 {
		return data
	}
	for i := start; i < end; i++ {
		data[i] = ' '
	}
	copy(data, data2)
	return data2
}
