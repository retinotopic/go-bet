package lobby

import (
	"strconv"
	"sync"
	"time"
)

type CustomImpl struct {
	*Game
	AdminOnce sync.Once
	Admin     *PlayUnit
}

func (c *CustomImpl) Validate(ctrl Ctrl) {
	c.AdminOnce.Do(func() {
		c.Admin = ctrl.Plr
	})
	if c.Admin == ctrl.Plr {
		//admin started the game, and also place 0 represents the table of cards itself
		if ctrl.Place == -1 {
			c.HasBegun = true
			c.Board.Bank = ctrl.CtrlBet
			sec, err := strconv.ParseInt(ctrl.Message, 10, 0)
			if err != nil {
				return
			}
			c.Board.Deadline = sec
			//at game start add occupied seats at the table to players
			c.AllUsers.Mtx.RLock()
			i := 0
			for _, val := range c.AllUsers.M {
				if i == 8 {
					break
				}
				if val.Place >= 0 {
					c.Players[i] = val
					i++
				}
			}
			c.AllUsers.Mtx.RUnlock()

			c.StartGameCh <- true
		} else { // admin approves player seat
			c.AllUsers.Mtx.RLock()
			c.AllUsers.M[ctrl.Plr.User_id]
			c.AllUsers.Mtx.RUnlock()
			if ok {
				pl.Place = ctrl.Place
				pl.cache = pl.StoreCache()
				c.BroadcastPlayer(ctrl.Plr, true)
			}
		}
	} else if ctrl.Place >= 0 && ctrl.Place <= 7 {
		c.AllUsers.Mtx.RLock()
		if !c.Seats[ctrl.Plr.Place] {
			WriteTimeout(time.Second*5, c.Admin.Conn, ctrl.Plr.StoreCache())
		}
		c.AllUsers.Mtx.RUnlock()
	} else if ctrl.Place == -2 && ctrl.Plr.Place >= 0 { // player leaves
		c.AllUsers.Mtx.RLock()
		pl, ok := c.AllUsers.M[ctrl.Plr.User_id]
		c.AllUsers.Mtx.RUnlock()
		if ok {
			pl.Place = ctrl.Place
			pl.cache = pl.StoreCache()
			c.BroadcastPlayer(ctrl.Plr, true)
		}
	}

}
