package lobby

import (
	"strconv"
	// "sync"
	"time"
	"unicode"

	json "github.com/bytedance/sonic"
)

func (c *Lobby) ValidateCustom(ctrl Ctrl) {
	c.AdminOnce.Do(func() {
		c.Admin = ctrl.Plr
	})
	if c.Admin == ctrl.Plr {
		//admin started the game, and also place -1 represents the GameBoard
		if ctrl.Place == -1 {

			c.HasBegun = true
			c.Board.Bank = ctrl.Ctrl
			sec, err := strconv.ParseInt(ctrl.Text, 10, 0)
			if err != nil && (sec > 3600 || sec < 0) && (ctrl.Ctrl > 10000000 || ctrl.Ctrl < 0) {
				return
			}
			c.Board.Deadline = sec
			//at game start add occupied seats at the table to players
			c.Players = c.Players[:0]
			c.MapUsers.Mtx.RLock()
			i := 0
			for _, pl := range c.MapUsers.M {
				if i == 8 {
					break
				}
				if c.AllUsers[pl].Place >= 0 {
					c.Players = append(c.Players, pl)
					c.AllUsers[pl].Bank = c.Board.Bank // give the player his starting stack
					i++
				}
			}
			c.MapUsers.Mtx.RUnlock()
			c.Seats = [8]Seats{} // to assert that game is custom
			c.StartGameCh <- true
		} else if ctrl.Ctrl < len([]rune(ctrl.Text)) { // admin approves player at the table
			c.MapUsers.Mtx.RLock()
			pl, ok := c.MapUsers.M[ctrl.Text[:ctrl.Ctrl]]
			// is user exist && if place index isnt out of range && if place is not occupied
			if ok && ctrl.Place >= 0 && ctrl.Place <= 7 && !c.Seats[ctrl.Place].isOccupied {
				name := ctrl.Text[ctrl.Ctrl:]
				if isAlphanumeric(name) {
					c.AllUsers[pl].Name = name
					c.Seats[ctrl.Place].isOccupied = true
					c.AllUsers[pl].Place = ctrl.Place
					c.AllUsers[pl].StoreCache()
					c.BroadcastPlayer(ctrl.Plr, true)
				}
			}
			c.MapUsers.Mtx.RUnlock()
		}
	} else if ctrl.Place >= 0 && ctrl.Place <= 7 {
		//player's request to be at the table
		b, err := json.Marshal(ctrl)
		if err != nil {
			return
		}
		c.AllUsers[c.Admin].Conn.Write(b)
	} else if ctrl.Place == -2 && c.AllUsers[ctrl.Plr].Place >= 0 { // player leaves
		c.MapUsers.Mtx.RLock()
		pl, ok := c.MapUsers.M[c.AllUsers[ctrl.Plr].User_id]
		c.Seats[ctrl.Place].isOccupied = false
		if ok {
			c.AllUsers[pl].Place = ctrl.Place
			c.AllUsers[pl].StoreCache()
			c.BroadcastPlayer(ctrl.Plr, true)
		}
		c.MapUsers.Mtx.RUnlock()
	}

}
func (r *Lobby) PlayerOutCustom(plr []PlayUnit, place int) {
	placemsg := []byte(`{"GameOverPlace":"` + strconv.Itoa(place) + `"}`)
	for i := range plr {
		plr[i].Place = -2
		defer plr[i].StoreCache()
		plr[i].Conn.Write(placemsg)
	}
}
func isAlphanumeric(s string) bool {
	for _, char := range s {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
			return false
		}
		if unicode.IsLetter(char) && !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')) {
			return false
		}
	}
	return true
}
