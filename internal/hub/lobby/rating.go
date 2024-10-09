package lobby

import (
	"math"
	"strconv"
	"time"
)

type RatingImpl struct {
	*Game
	q Queue
}

type Queue interface {
	PublishTask([]byte, int) error
	NewMessage(int, int) ([]byte, error)
}

func (r *RatingImpl) Validate(ctrl Ctrl) {
	timer := time.NewTimer(time.Second * time.Duration(ctrl.CtrlInt))
	for range timer.C {
		r.StartGameCh <- true
		return
	}
}
func (r *RatingImpl) PlayerOut(plr []PlayUnit, place int) {
	baseChange := 30
	middlePlace := float64(len(r.Players)+1) / 2
	for i := range plr {
		defer plr[i].Conn.CloseNow()
		rating := int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
		userid, err := strconv.Atoi(plr[i].User_id)
		if err != nil {
			return
		}
		data, err := r.q.NewMessage(userid, rating)
		if err != nil {
			return
		}
		err = r.q.PublishTask(data, 5)
		if err != nil {
			return
		}

		r.AllUsers.Mtx.Lock()
		delete(r.AllUsers.M, plr[i].User_id)
		r.AllUsers.Mtx.Unlock()

		WriteTimeout(time.Second*5, plr[i].Conn, []byte(`{"GameOverPlace":"`+strconv.Itoa(place)+`"}`))
	}
	if place == 1 {
		for range 3 {
			r.Shutdown <- true
		}
	}
}
