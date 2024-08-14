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
	c        Consume
	addr     string
	qD       QueueDeclare
	conn     *amqp.Connection
	ch       *amqp.Channel
	consume  <-chan amqp.Delivery
	queue    amqp.Queue
	TaskFunc func(int, int) error
}

func (t *TaskQueue) Close() {
	t.ch.Close()
	t.conn.Close()
}
func (t *TaskQueue) TryConnect() {
	conn, err := amqp.Dial(t.addr)
	if err != nil {
		panic(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	t.ch = ch
	q, err := t.ch.QueueDeclare(
		t.qD.Name,
		t.qD.Durable,
		t.qD.AutoDelete,
		t.qD.Exclusive,
		t.qD.NoWait,
		t.qD.Args,
	)
	if err != nil {
		panic(err)
	}
	t.queue = q
	consume, err := t.ch.Consume(
		t.queue.Name,
		t.c.Consumer,
		t.c.AutoAck,
		t.c.Exclusive,
		t.c.NoLocal,
		t.c.NoWait,
		t.c.Args,
	)
	if err != nil {
		panic(err)
	}
	err = t.ch.Confirm(false)
	if err != nil {
		panic(err)
	}
	t.consume = consume
	t.processConsume()
}

func NewQueue(addr string, c Consume, qD QueueDeclare, taskFunc func(int, int) error) *TaskQueue {
	t := &TaskQueue{}
	t.addr = addr
	t.c = c
	t.qD = qD
	t.TaskFunc = taskFunc
	return t
}
