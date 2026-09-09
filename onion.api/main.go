package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"onion.api/aplication"
	"onion.api/infrastrucre"
	"onion.api/persistence"
	"onion.api/presentation"
	"onion.api/utils"
)

func main() {
	environmentVariables := utils.LoadEnvironmentVariables()

	persistenceModule, error := persistence.Initialize(environmentVariables.PersistenceSettings)
	if error != nil {
		fmt.Printf("Error initializing persistence layer: %v\n", error)
		os.Exit(1)
	}
	defer persistenceModule.Close()

	infrastructureModule, error := infrastrucre.Initialize(environmentVariables.InfrastructureSettings, persistenceModule)
	if error != nil {
		fmt.Printf("Error initializing infrastructure layer: %v\n", error)
		os.Exit(1)
	}

	applicationModule := aplication.Initialize(persistenceModule, infrastructureModule)

	presentationModule := presentation.Initialize(environmentVariables.PresentationSettings, applicationModule)

	go func() {
		error := presentationModule.Start(environmentVariables.PresentationSettings)
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
