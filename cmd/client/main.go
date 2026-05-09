package main

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	fmt.Println("Starting Peril client...")
	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		fmt.Println("Failed to connect to RabbitMQ")
		panic(err)
	}
	defer conn.Close()
	fmt.Println("Successfully connected to RabbitMQ")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Failed to get user name")
		panic(err)
	}
	gamestate := gamelogic.NewGameState(userName)
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		"pause."+userName,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gamestate),
	)
	if err != nil {
		fmt.Println("Failed to declare and bind pause queue")
		panic(err)
	}
	mvQueueStr := "army_moves." + userName
	mvRoutingKey := "army_moves.*"
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		mvQueueStr,
		mvRoutingKey,
		pubsub.Transient,
		handlerArmyMoves(gamestate),
	)

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		if words[0] == "spawn" {
			err = gamestate.CommandSpawn(words)
			if err != nil {
				fmt.Println("Failed to spawn unit", err)
				continue
			}

			continue
		}
		if words[0] == "move" {
			if len(words) != 3 {
				fmt.Println("Usage: move <location> <unitID>")
				continue
			}
			am, err := gamestate.CommandMove(words)
			if err != nil {
				fmt.Println("Failed to move unit", err)
				continue
			}
			// publish move
			ch, err := conn.Channel()
			if err != nil {
				fmt.Println("Failed to open channel", err)
				continue
			}
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				mvRoutingKey,
				am,
			)
			if err != nil {
				fmt.Println("Failed to publish move", err)
				continue
			}
			fmt.Println("Moved unit.")
			continue
		}
		if words[0] == "status" {
			gamestate.CommandStatus()
			continue
		}
		if words[0] == "spam" {
			fmt.Println("Spamming not allowed yet!")
			continue
		}
		if words[0] == "help" {
			gamelogic.PrintClientHelp()
			continue
		}
		if words[0] == "quit" {
			gamelogic.PrintQuit()
			os.Exit(0)
		}
		fmt.Println("Unknown command")
	}

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Shutting down Peril client...")
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ") // print a prompt when the function returns
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerArmyMoves(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ") // print a prompt when the function returns
		mo := gs.HandleMove(am)
		if mo == gamelogic.MoveOutComeSafe || mo == gamelogic.MoveOutcomeMakeWar {
			return pubsub.Ack
		}
		return pubsub.NackDiscard
	}
}
