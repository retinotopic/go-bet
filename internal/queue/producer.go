package queue

import (
	json "github.com/bytedance/sonic"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Exchange struct {
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Durable    bool       `json:"durable"`
	AutoDelete bool       `json:"autoDelete"`
	Internal   bool       `json:"internal"`
	NoWait     bool       `json:"noWait"`
	Args       amqp.Table `json:"args"`
}

type Producer struct {
	Ex         Exchange
	RoutingKey string
	Addr       string
	ch         *amqp.Channel
	TaskFunc   func(int, int) error
}

func (p *Producer) TryConnect() {
	conn, err := amqp.Dial(p.Addr)
	if err != nil {
		panic(err)
	}
	p.ch, err = conn.Channel()
	if err != nil {
		panic(err)
	}
	err = p.ch.ExchangeDeclare(
		p.Ex.Name,
		p.Ex.Kind,
		p.Ex.Durable,
		p.Ex.AutoDelete,
		p.Ex.Internal,
		p.Ex.NoWait,
		p.Ex.Args,
	)

	if err != nil {
		panic(err)
	}
	err = p.ch.Confirm(false)
	if err != nil {
		panic(err)
	}

}
func (p *Producer) PublishTask(data []byte, retry int) error {

	confirmation, err := p.ch.PublishWithDeferredConfirm(
		p.Ex.Name,    // exchange
		p.RoutingKey, // routing key
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
	ok := confirmation.Wait()
	if !ok {
		if retry <= 1 {
			return err
		}
		return p.PublishTask(data, retry-1)
	}
	return nil
}
func (p *Producer) NewMessage(user_id, rating int) ([]byte, error) {
	m := &Message{User_id: user_id, Rating: rating}
	data, err := json.Marshal(m)
	return data, err
}
