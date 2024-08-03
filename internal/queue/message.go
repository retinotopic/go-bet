package queue

import (
	"github.com/goccy/go-json"
)

type Message struct {
	User_id  int `json:"user_id"`
	Rating   int `json:"rating"`
	Attempts int `json:"attempts"`
}

func (t *TaskQueue) NewMessage(user_id, rating int) ([]byte, error) {
	m := &Message{User_id: user_id, Rating: rating}
	data, err := json.Marshal(m)
	return data, err
}
