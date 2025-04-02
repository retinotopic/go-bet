package lobby

import (
	"math"
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
func (l *Lobby) PlayerOutRating(plr []PlayUnit, place int) {
	baseChange := 30
	placemsg := []byte(`{"GameOverPlace":"` + strconv.Itoa(place) + `"}`)
	middlePlace := float64(len(l.Players)+1) / 2
	for i := range plr {
		rating := int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
		userid, err := strconv.Atoi(plr[i].User_id)
		if err != nil {
			return
		}
		l.UpdateRating(userid, rating)
		plr[i].Place = -2
		defer plr[i].StoreCache()

		go WriteTimeout(time.Second*5, plr[i].Conn, placemsg)

	}
	if place == 1 {
		for range 3 {
			l.Shutdown <- true
		}
	}
}
