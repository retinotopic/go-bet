package lobby

import (
	"sync"
	"time"

	"github.com/chehsunliu/poker"
	"github.com/retinotopic/go-bet/pkg/wsutils"
)

type CustomImpl struct {
	*Game
	AdminOnce sync.Once
	Admin     *PlayUnit
}

func (c *CustomImpl) Validate(ctrl Ctrl) {
	if c.HasBegun {
		//when the game is in progress, a validation is made to ensure that the bet itself cannot exceed the bank
		if ctrl.CtrlBet <= ctrl.plr.Bankroll && ctrl.CtrlBet >= 0 {
			if ctrl.plr.Place > 0 {
				ctrl.plr.CtrlBet = ctrl.CtrlBet
				ctrl.plr.ExpirySec = 30
				c.PlayerCh <- ctrl.plr
			} else if ctrl.plr.Place == 0 {
				c.HasBegun = false
				//if the game is in progress and the playunit of the board itself comes into the channel, then the game is completely finished
			}
		}
	} else {
		if c.Admin == ctrl.plr {
			//admin started the game, and also place 0 represents the table of cards itself
			if ctrl.Place == 0 {
				c.HasBegun = true
				i := 0
				//at game start add occupied seats at the table to players
				c.m.Range(func(key string, val *PlayUnit) (stop bool) {
					if val.Place > 0 {
						c.Players[i] = val
						i++
					}
					return
				})
				c.StartGameCh <- true
			} else {
				pl, ok := c.m.Load(ctrl.plr.User_id) // admin approves player seat
				if ok {
					pl.Place = ctrl.Place
					c.BroadcastCh <- ctrl.plr
				}
			}
		} else if ctrl.Place > 0 && ctrl.Place <= 8 {
			//player's request to be at the table
			c.Admin.Conn.WriteJSON(ctrl.plr)
		} else if ctrl.Place == -1 {
			//player leaves the table
			pl, ok := c.m.Load(ctrl.plr.User_id)
			if ok {
				pl.Place = ctrl.Place
				c.BroadcastCh <- ctrl.plr
			}
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
	plr.Conn.WriteJSON(c.Board)
	for _, v := range c.Players { // load current state of the game
		pb := *v
		pb.Cards = []poker.Card{}
		plr.Conn.WriteJSON(pb)

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
