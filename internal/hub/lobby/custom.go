package lobby

import (
	"sync"

	"github.com/chehsunliu/poker"
)

type CustomImpl struct {
	*Game
	AdminOnce sync.Once
	Admin     *PlayUnit
}

func (c *CustomImpl) Validate(ctrl Ctrl) {
	if c.Admin == ctrl.Plr {
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
			pl, ok := c.m.Load(ctrl.Plr.User_id) // admin approves player seat
			if ok {
				pl.Place = ctrl.Place
				c.BroadcastCh <- ctrl.Plr
			}
		}
	} else if ctrl.Place > 0 && ctrl.Place <= 8 {
		//player's request to be at the table
		c.Admin.Conn.WriteJSON(ctrl.Plr)
	} else if ctrl.Place == -1 {
		//player leaves the table
		pl, ok := c.m.Load(ctrl.Plr.User_id)
		if ok {
			pl.Place = ctrl.Place
			c.BroadcastCh <- ctrl.Plr
		}
	}

}
func (c *CustomImpl) HandleConn(plr *PlayUnit) {
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
		ctrl.Plr = plr
		plr.IsAway = false
		c.PlayerCh <- ctrl
	}
}
