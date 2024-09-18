package lobby

import (
	"sync"
	"time"

	"github.com/retinotopic/go-bet/pkg/wsutils"
)

type CustomImpl struct {
	*Game
	AdminOnce sync.Once
	Admin     *PlayUnit
}

func (c *CustomImpl) Validate(ctrl Ctrl) {
	if c.HasBegun {
		if ctrl.CtrlBet <= ctrl.plr.Bankroll && ctrl.CtrlBet >= 0 {
			if ctrl.plr.Place > 0 {
				ctrl.plr.CtrlBet = ctrl.CtrlBet
				ctrl.plr.ExpirySec = 30
				c.PlayerCh <- ctrl.plr
			} else if ctrl.plr.Place == 0 {
				c.HasBegun = false
			}
		}
	} else {
		if c.Admin == ctrl.plr {
			if ctrl.Place == 0 {
				c.HasBegun = true
				i := 0
				c.m.Range(func(key string, val *PlayUnit) (stop bool) { // spectators
					if val.Place > 0 {
						c.Players[i] = val
						i++
					}
					return
				})
				c.StartGameCh <- true
			} else {
				pl, ok := c.m.Load(ctrl.User_id) // admin approves player seat
				if ok {
					pl.Place = ctrl.Place
					c.BroadcastCh <- ctrl.plr
				}
			}
		} else if ctrl.Place > 0 && ctrl.Place <= 8 {
			c.Admin.Conn.WriteJSON(ctrl.plr)
		}
	}

}
func (c *CustomImpl) HandleConn(plr *PlayUnit) {
	go wsutils.KeepAlive(plr.Conn, time.Second*15)
	c.m.SetIfAbsent(plr.User_id, plr)

	c.AdminOnce.Do(func() {
		c.Admin = plr
	})
	defer func() {
		if c.m.DeleteIf(plr.User_id, func(value *PlayUnit) bool { return value.Place == -1 }) { //if the seat is unoccupied, delete from map
			c.BroadcastCh <- plr
		}
		plr.Conn.Close()
	}()

	for _, v := range c.Players { // load current state of the game
		protect := false
		if v.Place != plr.Place {
			protect = true
		}
		plr.Send(v, protect)
	}
	for { // listening for actions
		ctrl := Ctrl{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			break
		}
		ctrl.plr = plr
		c.ValidateCh <- ctrl
	}
}
