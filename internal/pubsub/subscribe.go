package pubsub

import (
	"fmt"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, // an enum to represent "durable" or "transient"
    handler func(T),
) error {
	ch, _, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err!=nil {
		return err
	}
	delChan, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for msg := range delChan {
			var val T
			err := json.Unmarshal(msg.Body, &val)
			if err != nil {
				fmt.Println("Failed to unmarshal message")
				continue
			}
			handler(val)
			msg.Ack(false)
		}
	}()
	return nil
}