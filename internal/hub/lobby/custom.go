package lobby

import (
	"strconv"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
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
		//admin started the game, and also place -1 represents the table of cards itself
		if ctrl.Place == -1 {
			c.HasBegun = true
			c.Board.Bank = ctrl.CtrlInt
			sec, err := strconv.ParseInt(ctrl.CtrlString, 10, 0)
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
			pl, ok := c.AllUsers.M[ctrl.CtrlString]

			c.AllUsers.Mtx.RUnlock()
			// is user exist && if place index isnt out of range && if place is not occupied
			if ok && ctrl.Place >= 0 && ctrl.Place <= 7 && !c.Seats[ctrl.Place].isOccupied {
				c.Seats[ctrl.Place].isOccupied = true
				pl.Place = ctrl.Place
				pl.cache = pl.StoreCache()
				c.BroadcastPlayer(ctrl.Plr, true)
			}
		}
	} else if ctrl.Place >= 0 && ctrl.Place <= 7 {
		//player's request to be at the table
		b, err := json.Marshal(ctrl)
		if err != nil {
			return
		}
		WriteTimeout(time.Second*5, c.Admin.Conn, b)
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
