package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"onion.api/infrastrucre"
	"onion.api/persistence"
	"onion.api/presentation"
	"onion.api/utils"
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

	infrastructureSettings := infrastrucre.InfrastructureSettings{
		PakNSave: infrastrucre.RetailerSettings{
			ReferenceStore:      environmentVariables.PakNSave.ReferenceStore,
			DegreeOfParallelism: environmentVariables.PakNSave.DegreeOfParallelism,
			NumOfRetries:        environmentVariables.PakNSave.NumOfRetries,
			DelayInMs:           environmentVariables.PakNSave.DelayInMs,
			DelayMaxInMs:        environmentVariables.PakNSave.DelayMaxInMs,
			TimeoutInMs:         environmentVariables.PakNSave.TimeoutInMs,
		},
		Woolworths: infrastrucre.RetailerSettings{
			ReferenceStore:      environmentVariables.Woolworths.ReferenceStore,
			DegreeOfParallelism: environmentVariables.Woolworths.DegreeOfParallelism,
			NumOfRetries:        environmentVariables.Woolworths.NumOfRetries,
			DelayInMs:           environmentVariables.Woolworths.DelayInMs,
			DelayMaxInMs:        environmentVariables.Woolworths.DelayMaxInMs,
			TimeoutInMs:         environmentVariables.Woolworths.TimeoutInMs,
		},
	}

	infrastructureModule, error := infrastrucre.Initialize(infrastructureSettings)
	if error != nil {
		fmt.Printf("Error initializing infrastructure layer: %v\n", error)
		os.Exit(1)
	}

	presentationSettings := presentation.PresentationSettings{}
	presentationModule := presentation.Initialize(presentationSettings, infrastructureModule)

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
