package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueDeclare struct {
	Name       string     `json:"name"`
	Durable    bool       `json:"durable"`
	AutoDelete bool       `json:"autoDelete"`
	Exclusive  bool       `json:"exclusive"`
	NoWait     bool       `json:"noWait"`
	Args       amqp.Table `json:"args"`
}

type Consume struct {
	Consumer  string     `json:"consumer"`
	AutoAck   bool       `json:"autoAck"`
	Exclusive bool       `json:"exclusive"`
	NoLocal   bool       `json:"noLocal"`
	NoWait    bool       `json:"noWait"`
	Args      amqp.Table `json:"args"`
}
type Config struct {
	QueueDeclare QueueDeclare `json:"queueDeclare"`
	Consume      Consume      `json:"consume"`
}
type TaskQueue struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	consume <-chan amqp.Delivery
	queue   amqp.Queue
}

func (t *TaskQueue) Close() {
	t.ch.Close()
	t.conn.Close()
}

func DeclareAndRun(addr string, c Consume, qD QueueDeclare) (*TaskQueue, error) {
	t := &TaskQueue{}
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	t.ch = ch
	q, err := t.ch.QueueDeclare(
		qD.Name,
		qD.Durable,
		qD.AutoDelete,
		qD.Exclusive,
		qD.NoWait,
		qD.Args,
	)
	if err != nil {
		return nil, err
	}
	t.queue = q
	consume, err := t.ch.Consume(
		t.queue.Name,
		c.Consumer,
		c.AutoAck,
		c.Exclusive,
		c.NoLocal,
		c.NoWait,
		c.Args,
	)
	if err != nil {
		return nil, err
	}
	t.consume = consume
	t.processConsume()
	return t, err
}
