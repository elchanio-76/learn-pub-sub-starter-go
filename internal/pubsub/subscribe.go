package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	ch, _, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err != nil {
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
			ack := handler(val)
			if ack == NackRequeue {
				msg.Nack(false, true)
				//log.Println("Requeuing message")
				continue
			} else if ack == NackDiscard {
				msg.Nack(false, false)
				//log.Println("Discarding message")
				continue
			}
			msg.Ack(false)
			//log.Println("Acknowledged message")
		}
	}()
	return nil
}
