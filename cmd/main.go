package main

import (
	"log"
	"os"
	"zodGeneratorCLI/internal/controllers"
	"zodGeneratorCLI/internal/handlers"
)

func main() {
	controller := controllers.NewController()
	if err := handlers.NewHandler(controller).Run(os.Args); err != nil {
		log.Fatal(err.Error())
		return
	}
}
