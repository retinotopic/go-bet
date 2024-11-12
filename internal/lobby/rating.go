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
	timer := time.NewTimer(time.Minute * time.Duration(ctrl.Ctrl)) //pre-set timer
	r.Board.Deadline = time.Now().Add(time.Minute * time.Duration(ctrl.Ctrl)).Unix()
	r.Board.Bank = 1000
	for i := range 8 {
		r.Seats[i].place = uint8(i)
	}
	for range timer.C {
		r.StartGameCh <- true
		return
	}
}
func (r *RatingImpl) PlayerOut(plr []*PlayUnit, place int) {
	baseChange := 30
	placemsg := []byte(`{"GameOverPlace":"` + strconv.Itoa(place) + `"}`)
	middlePlace := float64(len(r.Players)+1) / 2
	for i := range plr {
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
		plr[i].Place = -2
		defer plr[i].StoreCache()

		go WriteTimeout(time.Second*5, plr[i].Conn, placemsg)

	}
	if place == 1 {
		for range 3 {
			r.Shutdown <- true
		}
	}
}
