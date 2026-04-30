package bus

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbit struct {
	ch *amqp.Channel
}

func NewRabbitBroker(channel *amqp.Channel, queues []string) (Driver, error) {
	for _, queue := range queues {
		_, err := channel.QueueDeclare(queue, true, true, false, false, nil)
		if err != nil {
			return nil, err
		}
	}
	return rabbit{ch: channel}, nil
}

func (r rabbit) Publish(queue string, data []byte) error {
	return r.ch.Publish("", queue, false, false, amqp.Publishing{
		ContentType: "application/octet-stream",
		Body:        data,
		Timestamp:   time.Now(),
	})
}

func (r rabbit) Subscribe(queue string) (<-chan []byte, <-chan error, error) {
	data, err := r.ch.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		return nil, nil, err
	}

	dataCh := make(chan []byte)
	go func() {
		defer close(dataCh)
		for d := range data {
			dataCh <- d.Body
		}
	}()
	errCh := make(chan error)
	close(errCh)
	return dataCh, errCh, nil
}
