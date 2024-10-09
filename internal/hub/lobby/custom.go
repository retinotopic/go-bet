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
			if err != nil && (sec > 3600 || sec < 0) && (ctrl.CtrlInt < 10000000 || ctrl.CtrlInt > 0) {
				return
			}
			c.Board.Deadline = sec
			//at game start add occupied seats at the table to players
			c.Players = make([]*PlayUnit, 0, 8)
			c.AllUsers.Mtx.RLock()
			i := 0
			for _, pl := range c.AllUsers.M {
				if i == 8 {
					break
				}
				if pl.Place >= 0 {
					c.Players = append(c.Players, pl)
					pl.Bank = c.Board.Bank // give the player his starting stack
					i++
				}
			}
			c.AllUsers.Mtx.RUnlock()
			c.Seats = [8]Seats{} // to assert that game is custom
			c.StartGameCh <- true
		} else { // admin approves player seat
			c.AllUsers.Mtx.RLock()
			pl, ok := c.AllUsers.M[ctrl.CtrlString]

			c.AllUsers.Mtx.RUnlock()
			// is user exist && if place index isnt out of range && if place is not occupied
			if ok && ctrl.Place >= 0 && ctrl.Place <= 7 && !c.Seats[ctrl.Place].isOccupied {
				c.Seats[ctrl.Place].isOccupied = true
				pl.Place = ctrl.Place
				pl.StoreCache()
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
func (r *CustomImpl) PlayerOut(plr []PlayUnit, place int) {
	for i := range plr {
		plr[i].Place = -2
		defer plr[i].StoreCache()
		go WriteTimeout(time.Second*5, plr[i].Conn, []byte(`{"GameOverPlace":"`+strconv.Itoa(place)+`"}`))
	}

}
