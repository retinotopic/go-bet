package queue

func (t *TaskQueue) ConsumeQueue(queueName string) error {
	consume, err := t.Ch.Consume(
		t.Queue.Name, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return err
	}
	t.Consume = consume
	return nil
}
