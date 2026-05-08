package main

import (
	"fmt"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
)

func main() {
	fmt.Println("Starting Peril server...")

	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		fmt.Println("Failed to connect to RabbitMQ")
		panic(err)
	}
	defer conn.Close()
	fmt.Println("Successfully connected to RabbitMQ")
	gamelogic.PrintServerHelp()
	
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println("Failed to open a channel")
		panic(err)
	}
	defer ch.Close()
	
	err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})
	if err != nil {
		fmt.Println("Failed to publish message")
		panic(err)
	}
	// Declare and bind a queue to the exchange for game logs
	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		"game_logs",
		"game_logs.*",
		pubsub.Durable,
	)

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		if words[0]== "pause" {
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
			if err != nil {
				fmt.Println("Failed to publish message")
				panic(err)
			}
			fmt.Println("Sent pause message")
			continue
		}
		if words[0] == "resume" {
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
			if err != nil {
				fmt.Println("Failed to publish message")
				panic(err)
			}
			fmt.Println("Sent resume message")
			continue
		}
		if words[0] == "quit" {
			break
		}
		fmt.Println("Unknown command")
	}

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Shutting down Peril server...")

}
