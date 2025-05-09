package lobby

import (
	"strconv"
	"time"
)

func (l *Lobby) ValidateRating(ctrl Ctrl) {
	l.TurnTimer.Reset(time.Minute * time.Duration(ctrl.Ctrl)) //pre-set timer
	l.Board.Deadline = time.Now().Add(time.Minute * time.Duration(ctrl.Ctrl)).Unix()
	l.Board.Bank = 1000
	for i := range 8 {
		l.Seats[i].place = uint8(i)
	}
	for range l.TurnTimer.C {
		l.StartGameCh <- true
		return
	}
}
func (l *Lobby) PlayerOut(plr []int, place int, shutdown bool) {
	placemsg := []byte(`{"GameOverPlace":"` + strconv.Itoa(place) + `"}`)
	for i := range plr {
		l.AllUsers[plr[i]].Place = -2
		defer l.AllUsers[plr[i]].StoreCache()
		l.AllUsers[plr[i]].Conn.Write(placemsg)
	}
	if shutdown {
		l.Shutdown = true
	}
}
