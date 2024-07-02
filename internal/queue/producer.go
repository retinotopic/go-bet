package queue

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (t *TaskQueue) PublishTask(task TaskMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}
	confirmation, err := t.Ch.PublishWithDeferredConfirm(
		"",           // exchange
		t.Queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
	if err != nil {
		return err
	}
	ok, err := confirmation.WaitContext(ctx)
	if !ok || err != nil {
		return t.PublishTask(task)
	}
	return nil
}
