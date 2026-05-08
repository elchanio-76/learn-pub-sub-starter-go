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
		fmt.Println("Failed to declare and bind queue")
		panic(err)
	}

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
			/*
				err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.SpawnKey, routing.SpawnMessage{
					UnitType: unitType,
					Location: location,
				})
				if err != nil {
					fmt.Println("Failed to publish message")
					panic(err)
				}
				fmt.Println("Sent spawn message")
			*/
			continue
		}
		if words[0] == "move" {
			if len(words) != 3 {
				fmt.Println("Usage: move <location> <unitID>")
				continue
			}
			_, err = gamestate.CommandMove(words)
			if err != nil {
				fmt.Println("Failed to move unit", err)
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


func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ") // print a prompt when the function returns
		gs.HandlePause(ps)
	}
}

