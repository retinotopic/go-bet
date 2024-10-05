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
	timer := time.NewTimer(time.Second * 45)
	for range timer.C {
		r.StartGameCh <- true
		return
	}
}
func (r *RatingImpl) PlayerOut(plr []PlayUnit, place int) {
	baseChange := 30
	middlePlace := float64(len(r.Players)+1) / 2
	for i := range plr {
		rating := int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
		userid, err := strconv.Atoi(plr[i].User_id)
		data, err := r.q.NewMessage(userid, rating)
		if err != nil {
			r.q.PublishTask(data, 5)
		}

		writeTimeout(time.Second*5, plr[i].Conn, []byte(`{"GameOverPlace":"`+strconv.Itoa(place)+`"}`))
		plr[i].Conn.CloseNow()
		r.MapTable.Delete(plr[i].User_id)
	}
	if place == 1 {
		for range 3 {
			r.Shutdown <- true
		}
	}
}
