package main

import (
	"fmt"
	"log"
	"time"

	"finance-backend/config"
	"finance-backend/routes"

	"github.com/gin-contrib/cors"
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
	checkErrOrPrint(msg, err)

	msg, err = MessageFormater(Yellow, "loading environment variables...")
	checkErrOrPrint(msg, err)
	config.LoadEnv()

	msg, err = MessageFormater(Yellow, "connecting to database...")
	checkErrOrPrint(msg, err)

	transactionsPath := config.GetEnv("TRANSACTIONS_DB_PATH")
	_, err = config.ConnectDB("transactions", transactionsPath)
	if err != nil {
		log.Fatal(MessageFormaterMust(Red, "Error trying to connect to transactions table: "+err.Error()))
	}

	cardsPath := config.GetEnv("CARDS_DB_PATH")
	_, err = config.ConnectDB("cards", cardsPath)
	if err != nil {
		log.Fatal(MessageFormaterMust(Red, "Error trying to connect to cards table: "+err.Error()))
	}

	msg, err = MessageFormater(Yellow, "setting routes...")
	checkErrOrPrint(msg, err)
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// ✅ CORS global habilitado (desarrollo o API pública)
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	/*
		// 🔒 CORS restringido (para producción)
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"https://casapipis.net"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	*/

	routes.SetupRoutes(r)

	port := config.GetEnv("PORT")
	portMsg := "Trying to serve HTTP on port..." + port
	msg, err = MessageFormater(Cyan, portMsg)
	checkErrOrPrint(msg, err)

	r.Run("0.0.0.0:" + port) // ✅ escucha en todas las interfaces

}

func MessageFormater(color Color, message string) (string, error) {
	val, ok := colorMap[color]
	if ok {
		formatted := "\033[" + val + message + "\033[0m"
		return formatted, nil
	}
	return "", fmt.Errorf("error: color not found at MessageFormater()")
}

func MessageFormaterMust(color Color, message string) string {
	formatted, _ := MessageFormater(color, message)
	return formatted
}

func checkErrOrPrint(msg string, err error) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg)
}
