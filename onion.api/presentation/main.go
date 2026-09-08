package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"onion.api/persistence"
)

func main() {
	loadEnvironmentVariables()

	databaseConnectionString := os.Getenv("DATABASE_URL")
	if databaseConnectionString == "" {
		fmt.Println("Error: DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	persistenceSettings := persistence.PersistenceSettings{
		ConnectionString: databaseConnectionString,
	}

	persistenceModule, error := persistence.Initialize(persistenceSettings)
	if error != nil {
		fmt.Printf("Error initializing persistence layer: %v\n", error)
		os.Exit(1)
	}
	defer persistenceModule.Close()

	fmt.Println("Application started successfully")

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	fmt.Println("Shutting down...")
}
