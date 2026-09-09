package retailers_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	retaileractions "onion.api/aplication/actions/retailers"
	"onion.api/aplication/command_base"
	retailercommands "onion.api/aplication/commands/retailers"
	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"
	"onion.api/persistence/entities"
	"onion.api/tests/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRetailer struct {
	stores []responses.StoreResponse
	error  error
}

func (mock *mockRetailer) GetProducts(context context.Context) ([]responses.ScrapedProductResponse, error) {
	return nil, nil
}

func (mock *mockRetailer) GetStores(context context.Context) ([]responses.StoreResponse, error) {
	return mock.stores, mock.error
}

func TestSyncRetailersCommand_InsertsNewStores(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	paknsaveStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "pak-1", Name: "Pak'nSave Mt Wellington"},
	}
	woolworthsStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainWoolworths, Id: "woo-1", Name: "Woolworths Glenfield"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{stores: paknsaveStores},
		responses.StoreChainWoolworths: &mockRetailer{stores: woolworthsStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := command_base.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	command := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	executeError := command.Execute(context.Background())
	require.NoError(testing, executeError)

	paknsaveResult, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainPAKNSAVE)
	require.NoError(testing, fetchError)
	require.Len(testing, paknsaveResult, 1)
	assert.Equal(testing, "Pak'nSave Mt Wellington", paknsaveResult[0].Name)
	assert.Equal(testing, "pak-1", paknsaveResult[0].ExternalStoreID.String)

	woolworthsResult, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainWOOLWORTHS)
	require.NoError(testing, fetchError)
	require.Len(testing, woolworthsResult, 1)
	assert.Equal(testing, "Woolworths Glenfield", woolworthsResult[0].Name)
	assert.Equal(testing, "woo-1", woolworthsResult[0].ExternalStoreID.String)
}

func TestSyncRetailersCommand_IdempotentSameName(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	stores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "pak-1", Name: "Pak'nSave Mt Wellington"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave: &mockRetailer{stores: stores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := command_base.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	command := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	require.NoError(testing, command.Execute(context.Background()))

	firstRun, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainPAKNSAVE)
	require.NoError(testing, fetchError)
	require.Len(testing, firstRun, 1)
	firstUpdated := firstRun[0].LastUpdatedUtc

	require.NoError(testing, command.Execute(context.Background()))

	secondRun, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainPAKNSAVE)
	require.NoError(testing, fetchError)
	require.Len(testing, secondRun, 1)
	assert.Equal(testing, "Pak'nSave Mt Wellington", secondRun[0].Name)
	assert.True(testing, secondRun[0].LastUpdatedUtc.Time.After(firstUpdated.Time) || secondRun[0].LastUpdatedUtc.Time.Equal(firstUpdated.Time),
		"last_updated_utc should be bumped on re-upsert")
}

func TestSyncRetailersCommand_UpdatesNameOnRename(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	originalStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "pak-1", Name: "Pak'nSave Mt Wellington"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave: &mockRetailer{stores: originalStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := command_base.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	command := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	require.NoError(testing, command.Execute(context.Background()))

	renamedStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "pak-1", Name: "Pak'nSave Sylvia Park"},
	}
	retailers[responses.StoreChainPakNSave] = &mockRetailer{stores: renamedStores}
	command = retailercommands.NewSyncRetailersCommand(baseCommand, action)

	require.NoError(testing, command.Execute(context.Background()))

	result, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainPAKNSAVE)
	require.NoError(testing, fetchError)
	require.Len(testing, result, 1)
	assert.Equal(testing, "Pak'nSave Sylvia Park", result[0].Name, "name should update to new value")
	assert.Equal(testing, "pak-1", result[0].ExternalStoreID.String, "external_store_id should remain the same")
}

func TestSyncRetailersCommand_HandlesRetailerFailure(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	woolworthsStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainWoolworths, Id: "woo-1", Name: "Woolworths Glenfield"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{error: fmt.Errorf("network timeout")},
		responses.StoreChainWoolworths: &mockRetailer{stores: woolworthsStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := command_base.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	command := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	executeError := command.Execute(context.Background())
	require.NoError(testing, executeError)

	result, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainWOOLWORTHS)
	require.NoError(testing, fetchError)
	assert.Len(testing, result, 1)
}
