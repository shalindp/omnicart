package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"onion.api/persistence"
	"onion.api/utils"

	"onion.api/presentation"
)

func main() {
	environmentVariables := utils.LoadEnvironmentVariables()

	persistenceSettings := persistence.PersistenceSettings{
		ConnectionString: environmentVariables.DatabaseURL,
	}

	persistenceModule, error := persistence.Initialize(persistenceSettings)
	if error != nil {
		fmt.Printf("Error initializing persistence layer: %v\n", error)
		os.Exit(1)
	}
	defer persistenceModule.Close()

	presentationSettings := presentation.PresentationSettings{}
	presentationModule := presentation.Initialize(presentationSettings)

	go func() {
		error := presentationModule.Start(":8080")
		if error != nil {
			fmt.Printf("Error starting server: %v\n", error)
			os.Exit(1)
		}
	}()

	fmt.Println("Application started on :8080")

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	fmt.Println("Shutting down...")
}
