package queue

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (t *TaskQueue) PublishTask(data []byte, retry int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	confirmation, err := t.ch.PublishWithDeferredConfirm(
		"",           // exchange
		t.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         data,
		})
	if err != nil {
		return err
	}
	ok, err := confirmation.WaitContext(ctx)
	if !ok || err != nil {
		if retry <= 1 {
			return err
		}
		return t.PublishTask(data, retry-1)
	}
	return nil
}
