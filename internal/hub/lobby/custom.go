package lobby

import (
	"sync"
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
			//at game start add occupied seats at the table to players
			c.m.Range(func(key string, val *PlayUnit) (stop bool) {
				if val.Place >= 0 {
					c.Players[i] = val
					i++
				}
				return
			})
			c.StartGameCh <- true
		} else {
			pl, ok := c.m.Load(ctrl.Plr.User_id) // admin approves player seat
			if ok {
				pl.Place = ctrl.Place
				c.BroadcastCh <- Ctrl{Plr: ctrl.Plr}
			}
		}
	} else if ctrl.Place >= 0 && ctrl.Place <= 7 {
		//player's request to be at the table
		c.Admin.Conn.WriteJSON(ctrl.Plr)
	} else if ctrl.Place == -2 && ctrl.Plr.Place >= 0 {
		//player leaves the table
		pl, ok := c.m.Load(ctrl.Plr.User_id)
		if ok {
			pl.Place = ctrl.Place
			c.BroadcastCh <- Ctrl{Plr: ctrl.Plr}
		}
	}

}
