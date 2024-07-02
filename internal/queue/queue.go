package queue

import (
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type TaskQueue struct {
	Conn    *amqp.Connection
	Ch      *amqp.Channel
	Consume <-chan amqp.Delivery
	Queue   amqp.Queue
}

var Queue TaskQueue

func init() {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("%v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("%v", err)
	}

	Queue = TaskQueue{Conn: conn, Ch: ch}
	err = Queue.DeclareQueue("task_queue")
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = Queue.ConsumeQueue()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func (t *TaskQueue) Close() {
	t.Ch.Close()
	t.Conn.Close()
}

func (t *TaskQueue) DeclareQueue(name string) error {
	q, err := t.Ch.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	t.Queue = q
	return nil
}
