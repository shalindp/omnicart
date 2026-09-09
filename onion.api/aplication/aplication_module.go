package aplication

import (
	"log"
	"os"

	retaileractions "onion.api/aplication/actions/retailers"
	"onion.api/aplication/command_base"
	retailercommands "onion.api/aplication/commands/retailers"
	"onion.api/infrastrucre"
	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"
	"onion.api/persistence"
)

type ApplicationModule struct {
	SyncRetailersCommand *retailercommands.SyncRetailersCommand
}

func Initialize(persistenceModule *persistence.PersistenceModule, infrastructureModule *infrastrucre.InfrastructureModule) *ApplicationModule {
	logger := log.New(os.Stdout, "[APP] ", log.LstdFlags)

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   infrastructureModule.PakNSaveClient,
		responses.StoreChainWoolworths: infrastructureModule.WoolworthsClient,
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := command_base.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	syncCommand := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	return &ApplicationModule{
		SyncRetailersCommand: syncCommand,
	}
}
