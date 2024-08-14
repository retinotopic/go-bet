package queue

import (
	"errors"
	"log"

	"github.com/goccy/go-json"
)

func (t *TaskQueue) processConsume() {
	for {
		err := t.consumeReceive()
		if err != nil {
			log.Printf("Error in consume: %v. Attempting to reconnect...", err)
			t.TryConnect()
		}
	}
}

func (t *TaskQueue) consumeReceive() error {
	for msg := range t.consume {
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

		err = t.TaskFunc(m.User_id, m.Rating)
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
	return errors.New("consume channel closed")
}
