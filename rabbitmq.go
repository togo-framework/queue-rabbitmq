// Package queuerabbitmq is a togo queue backend backed by RabbitMQ (AMQP 0-9-1).
// Implements togo.Queue; takes over k.Queue when selected with
// `togo provider:use queue rabbitmq`.
//
//   RABBITMQ_URL  amqp URL (default amqp://guest:guest@localhost:5672/)
package queuerabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/togo-framework/providers"
	"github.com/togo-framework/togo"
)

var _ togo.Queue = (*rabbitQueue)(nil)

func init() {
	togo.RegisterProviderFunc("queue-rabbitmq", togo.PriorityLate+1, func(k *togo.Kernel) error {
		if providers.Active(k, providers.CapQueue, "memory") != "rabbitmq" {
			return nil
		}
		q, err := newRabbit(k)
		if err != nil {
			if k.Log != nil {
				k.Log.Error("queue-rabbitmq: connect failed", "err", err)
			}
			return nil
		}
		k.Queue = q
		if k.Log != nil {
			k.Log.Info("plugin active", "plugin", "queue-rabbitmq")
		}
		return nil
	})
}

type rabbitQueue struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func newRabbit(k *togo.Kernel) (*rabbitQueue, error) {
	url := providers.Value(k, providers.CapQueue, "rabbitmq", "url", "amqp://guest:guest@localhost:5672/", false)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &rabbitQueue{conn: conn, ch: ch}, nil
}

func (q *rabbitQueue) Handle(name string, h togo.QueueHandler) {
	if _, err := q.ch.QueueDeclare(name, true, false, false, false, nil); err != nil {
		return
	}
	msgs, err := q.ch.Consume(name, "", true, false, false, false, nil)
	if err != nil {
		return
	}
	go func() {
		for m := range msgs {
			var payload any
			_ = json.Unmarshal(m.Body, &payload)
			_ = h(context.Background(), payload)
		}
	}()
}

func (q *rabbitQueue) Dispatch(ctx context.Context, name string, payload any) error {
	if _, err := q.ch.QueueDeclare(name, true, false, false, false, nil); err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.ch.PublishWithContext(ctx, "", name, false, false, amqp.Publishing{ContentType: "application/json", Body: b})
}
