package lobby

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/retinotopic/go-bet/pkg/wsutils"
)

type RatingImpl struct {
	*Lobby
	q Queue
}

type Queue interface {
	PublishTask([]byte, int) error
	NewMessage(int, int) ([]byte, error)
}

func (r *RatingImpl) tickerTillGame() {
	timer := time.NewTimer(time.Second * 45)
	for range timer.C {
		r.StartGame <- true
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

func (r *RatingImpl) HandleConn(plr PlayUnit) {
	go wsutils.KeepAlive(plr.Conn, time.Second*15)

	defer func() {

		if plr.Conn != nil {
			err := plr.Conn.Close()
			if err != nil {
				log.Println(err, "error closing connection")
			}
		}
	}()

	for _, v := range l.Players { // load current state of the game
		if v.Place != plr.Place {
			v = v.PrivateSend()
		}
		err := plr.Conn.WriteJSON(v)
		if err != nil {
			fmt.Println(err, "WriteJSON start")
		}
		fmt.Println("start")
	}
	for {
		ctrl := PlayUnit{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			log.Println(err, "conn read error")
			break
		}
		if ctrl.CtrlBet <= plr.Bankroll && ctrl.CtrlBet >= 0 {
			plr.CtrlBet = ctrl.CtrlBet
			plr.ExpirySec = 30
			l.PlayerCh <- plr
		}
	}
}
