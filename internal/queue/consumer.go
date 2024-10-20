package queue

import (
	"errors"
	"log"

	"github.com/goccy/go-json"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consume struct {
	Consumer  string     `json:"consumer"`
	AutoAck   bool       `json:"autoAck"`
	Exclusive bool       `json:"exclusive"`
	NoLocal   bool       `json:"noLocal"`
	NoWait    bool       `json:"noWait"`
	Args      amqp.Table `json:"args"`
}
type Consumer struct {
	Ex         Exchange
	RoutingKey string
	Addr       string
	ch         *amqp.Channel
	consume    <-chan amqp.Delivery
	TaskFunc   func(int, int) error
}

func (c *Consumer) TryConnect() {
	conn, err := amqp.Dial(c.Addr)
	if err != nil {
		panic(err)
	}
	c.ch, err = conn.Channel()
	if err != nil {
		panic(err)
	}
	err = c.ch.ExchangeDeclare(
		c.Ex.Name,
		c.Ex.Kind,
		c.Ex.Durable,
		c.Ex.AutoDelete,
		c.Ex.Internal,
		c.Ex.NoWait,
		c.Ex.Args,
	)
	if err != nil {
		panic(err)
	}
	q, err := c.ch.QueueDeclare(
		"",
		c.Ex.Durable,
		c.Ex.AutoDelete,
		false,
		c.Ex.NoWait,
		c.Ex.Args,
	)
	if err != nil {
		panic(err)
	}
	err = c.ch.QueueBind(
		q.Name,       // queue name
		c.RoutingKey, // routing key
		c.Ex.Name,    // exchange
		false,
		nil)

	if err != nil {
		panic(err)
	}
	c.consume, err = c.ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	if err != nil {
		panic(err)
	}
	c.processConsume()
}

func (c *Consumer) processConsume() {
	for {
		err := c.consumeReceive()
		if err != nil {
			log.Printf("Error in consume: %v. Attempting to reconnect...", err)
			c.TryConnect()
		}
	}
}

func (c *Consumer) consumeReceive() error {
	for msg := range c.consume {
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

		err = c.TaskFunc(m.User_id, m.Rating)
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
