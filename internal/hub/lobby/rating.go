package lobby

import (
	"math"
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

func (r *RatingImpl) tickerTillGame() {
	timer := time.NewTimer(time.Second * 45)
	for range timer.C {
		r.StartGameCh <- true
		return
	}
}
func (r *RatingImpl) PlayerOut(plr []PlayUnit, place int) {
	baseChange := 30
	middlePlace := float64(len(r.Players)+1) / 2
	for _, plr := range plr {
		rating := int(math.Round(float64(baseChange) * (middlePlace - float64(place)) / (middlePlace - 1)))
		data, err := r.q.NewMessage(plr.User_id, rating)
		if err != nil {
			r.q.PublishTask(data, 5)
		}
		plr.Conn.WriteJSON(plr)
		plr.Conn.Close()
	}
}
