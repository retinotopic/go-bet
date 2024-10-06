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
			i := 0
			c.Board.Bank = ctrl.CtrlBet
			sec, err := strconv.ParseInt(ctrl.Message, 10, 0)
			if err != nil {
				return
			}
			c.Board.Deadline = sec
			//at game start add occupied seats at the table to players
			c.MapUsers.Mtx.RLock()
			for _, val := range c.MapUsers.M {
				if val.Place >= 0 {
					c.Players[i] = val
					i++
				}
			}
			c.MapUsers.Mtx.RUnlock()

			c.StartGameCh <- true
		} else {
			c.MapUsers.Mtx.RLock()
			pl, ok := c.MapUsers.M[ctrl.Plr.User_id]
			c.MapUsers.Mtx.RUnlock()
			if ok {
				pl.Place = ctrl.Place
				pl.cache = pl.StoreCache()
				c.BroadcastPlayer(ctrl.Plr, true)
			}
		}
	} else if ctrl.Place >= 0 && ctrl.Place <= 7 {
		//player's request to be at the table
		ctrl.Plr.cache = ctrl.Plr.StoreCache()
		WriteTimeout(time.Second*5, c.Admin.Conn, ctrl.Plr.GetCache())
	} else if ctrl.Place == -2 && ctrl.Plr.Place >= 0 { // player leaves
		c.MapUsers.Mtx.RLock()
		pl, ok := c.MapUsers.M[ctrl.Plr.User_id] // admin approves player seat
		c.MapUsers.Mtx.RUnlock()
		if ok {
			pl.Place = ctrl.Place
			pl.cache = pl.StoreCache()
			c.BroadcastPlayer(ctrl.Plr, true)
		}
	}

}
