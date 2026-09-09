package retailers

import (
	"context"
	"fmt"

	retaileractions "onion.api/aplication/actions/retailers"
	"onion.api/aplication/command_base"
	"onion.api/infrastrucre/mappers"
	"onion.api/persistence/entities"

	"github.com/jackc/pgx/v5"
)

type SyncRetailersCommand struct {
	command_base.BaseCommand
	Action *retaileractions.GetRetailersStoresAction
}

func NewSyncRetailersCommand(baseCommand command_base.BaseCommand, action *retaileractions.GetRetailersStoresAction) *SyncRetailersCommand {
	return &SyncRetailersCommand{
		BaseCommand: baseCommand,
		Action:      action,
	}
}

func (command *SyncRetailersCommand) Execute(requestContext context.Context) error {
	stores, fetchError := command.Action.FetchAllRetailerStores(requestContext)
	if fetchError != nil {
		return fetchError
	}

	writeError := command.PersistenceModule.InTransaction(requestContext, func(queries *entities.Queries, transaction pgx.Tx) error {
		for _, store := range stores {
			_, upsertError := queries.UpsertStore(requestContext, entities.UpsertStoreParams{
				Retailer:        mappers.MapStoreChain(store.Retailer),
				RegionID:        "default",
				ExternalStoreID: store.Id,
				Name:            store.Name,
			})
			if upsertError != nil {
				return fmt.Errorf("sync retailers: upsert store for %s: %w", store.Retailer, upsertError)
			}
		}
		return nil
	})
	if writeError != nil {
		return writeError
	}

	command.Logger.Printf("sync retailers: upserted %d stores", len(stores))
	return nil
}
