package lobby

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/retinotopic/go-bet/pkg/wsutils"
)

type CustomImpl struct {
	*Lobby
	AdminOnce sync.Once
	Admin     PlayUnit
}

func (l *Lobby) ConnHandle(plr PlayUnit) {
	go wsutils.KeepAlive(plr.Conn, time.Second*15)
	l.AdminOnce.Do(func() {
		if l.IsRating {
			go l.tickerTillGame()
		} else {
			l.Admin = plr
			plr.Admin = true
		}
	})

	defer func() {
		if plr.Conn != nil {
			err := plr.Conn.Close()
			if err != nil {
				log.Println(err, "error closing connection")
			}
		}
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
		ctrl := PlayUnit{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			fmt.Println(err, "conn read error")
			break
		}
		if ctrl.CtrlBet <= plr.Bankroll && ctrl.CtrlBet >= 0 {
			plr.CtrlBet = ctrl.CtrlBet
			plr.ExpirySec = 30
			l.PlayerCh <- plr
		}
	}
}
