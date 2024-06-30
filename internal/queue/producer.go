package queue

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (t *TaskQueue) PublishTask(queueName string, task interface{}) error {
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return t.Ch.PublishWithContext(
		context.Background(),
		"",           // exchange
		t.Queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
}
