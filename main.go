package main

import (
	"fmt"
	"log"

	"finance-backend/config"
	"finance-backend/routes"

	"github.com/gin-gonic/gin"
)

// Define a new type based on string
type Color string

// Declare constants of that type for allowed values
const (
	Green  Color = "green"
	Red    Color = "red"
	Yellow Color = "yellow"
	Blue   Color = "blue"
	Cyan   Color = "cyan"
)

// Optional: a map from Color to the ANSI code string
var colorMap = map[Color]string{
	Red:    "31m",
	Green:  "32m",
	Yellow: "33m",
	Blue:   "34m",
	Cyan:   "1;36m",
}

func main() {

	msg, err := MessageFormater(Yellow, "starting server...")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(msg)
	}

	msg, err = MessageFormater(Yellow, "loading enviroment variables...")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(msg)
	}

	config.LoadEnv()

	msg, err = MessageFormater(Yellow, "connecting to database...")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(msg)
	}

	transactionsPath := config.GetEnv("TRANSACTIONS_DB_PATH")
	_, err = config.ConnectDB("transactions", transactionsPath)

	if err != nil {
		fmt.Println(MessageFormater(Red, "Error trying to connect to transactions table"))
		log.Fatal(err)
	}

	cardsPath := config.GetEnv("CARDS_DB_PATH")
	_, err = config.ConnectDB("cards", cardsPath)

	if err != nil {
		fmt.Println(MessageFormater(Red, "Error trying to connect to cards table"))
		log.Fatal(err)
	}

	msg, err = MessageFormater(Yellow, "setting routes...")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(msg)
	}

	r := gin.Default()
	routes.SetupRoutes(r)

	var port string = config.GetEnv("PORT")

	portMessage := "Trying to serve HTTP on port..." + port
	portParameter := ":" + port
	msg, err = MessageFormater(Cyan, portMessage)

	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(msg)
	}

	r.Run(portParameter)
}

func MessageFormater(color Color, message string) (string, error) {
	val, ok := colorMap[color]
	if ok {
		var formatedMessage string = "\033[" + val + message + "\033[0m"
		return formatedMessage, nil
	} else {
		return "", fmt.Errorf("error color not found at MessageFormater()")
	}
}
