package lobby

import (
	"fmt"
	"sync"
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/retinotopic/go-bet/pkg/wsutils"
)
type CustomImpl struct {
	*Lobby
	AdminOnce sync.Once
	AdminId     string
	HasBegun bool
}
func (c *CustomImpl) Validate(plr PlayUnit) {

}
func (c *CustomImpl) HandleConn(plr PlayUnit) {
	go wsutils.KeepAlive(plr.Conn, time.Second*15)
	c.AdminOnce.Do(func() {
		c.AdminId = plr.User_id
	})
	defer func() {
		if c.m.DeleteIf(plr.User_id,func(value PlayUnit) bool {return value.Place == -1}) {
			c.BroadcastCh<-plr
		}
		plr.Conn.Close()
	}()

	for _, v := range c.Players { // load current state of the game
		if v.Place != plr.Place || !v.Exposed {
			v = v.PrivateSend()
		}
		plr.Conn.WriteJSON(v)
	}
	for {
		ctrl := PlayUnit{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			fmt.Println(err, "conn read error")
			break
		}
		if game
		if ctrl.CtrlBet <= plr.Bankroll && ctrl.CtrlBet >= 0 {
			plr.CtrlBet = ctrl.CtrlBet
			plr.ExpirySec = 30
			c.PlayerCh <- plr
		}
		if !Game
			::place
			if c.AdminId = plr.User_id
				::board place


	}
}



