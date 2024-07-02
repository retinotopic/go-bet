package queue

import (
	"github.com/goccy/go-json"
	"github.com/retinotopic/go-bet/internal/db"
)

func (t *TaskQueue) ConsumeQueue() error {
	consume, err := t.Ch.Consume(
		t.Queue.Name, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return err
	}
	t.Consume = consume
	return nil
}

func (t *TaskQueue) ProcessConsume() {
	for msg := range t.Consume {
		m := &Message{}
		err := json.Unmarshal(msg.Body, m)
		if err != nil {
			msg.Reject(false)
			continue
		}
		if m.Attempts >= 5 {
			msg.Reject(false)
			continue
		}

		err = db.ChangeRating(m.User_id, m.Rating)
		if err != nil {
			m.Attempts++
			newBody, err := json.Marshal(m)
			if err != nil {
				msg.Reject(false)
				continue
			}
			msg.Body = newBody
			msg.Nack(false, true)
			continue
		}
		msg.Ack(false)
	}

}
