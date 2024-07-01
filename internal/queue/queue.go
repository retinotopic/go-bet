package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/retinotopic/go-bet/internal/db"
)

type TaskQueue struct {
	Conn    *amqp.Connection
	Ch      *amqp.Channel
	Consume <-chan amqp.Delivery
	Queue   amqp.Queue
	Db      db.PgClient
}

func NewTaskQueue(url string) (*TaskQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &TaskQueue{Conn: conn, Ch: ch}, nil
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
