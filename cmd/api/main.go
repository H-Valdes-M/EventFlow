package main

import (
	"eventflow/cmd/app"
	"log"
)

// @title EventFlow API
// @version 1.0
// @description API de EventFlow.
// @host localhost:8080
// @BasePath /
func main() {
	application := app.NewApp()
	if err := application.Init(); err != nil {
		log.Fatal(err)
	}
	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
