package retailers_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"onion.api/aplication"
	retaileractions "onion.api/aplication/actions/retailers"
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

func TestSyncRetailersCommand_UpsertsStores(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	paknsaveStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "1", Name: "Pak'nSave Mt Wellington"},
	}
	woolworthsStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainWoolworths, Id: "2", Name: "Woolworths Glenfield"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{stores: paknsaveStores},
		responses.StoreChainWoolworths: &mockRetailer{stores: woolworthsStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := aplication.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	command := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	executeError := command.Execute(context.Background())
	require.NoError(testing, executeError)

	stores, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainPAKNSAVE)
	require.NoError(testing, fetchError)
	require.Len(testing, stores, 1)
	assert.Equal(testing, "default", stores[0].RegionID)

	woolworthsStoresResult, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainWOOLWORTHS)
	require.NoError(testing, fetchError)
	require.Len(testing, woolworthsStoresResult, 1)
}

func TestSyncRetailersCommand_IsIdempotent(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	stores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "1", Name: "Pak'nSave Mt Wellington"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave: &mockRetailer{stores: stores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := aplication.BaseCommand{
		PersistenceModule: persistenceModule,
		Logger:            logger,
	}
	command := retailercommands.NewSyncRetailersCommand(baseCommand, action)

	require.NoError(testing, command.Execute(context.Background()))
	require.NoError(testing, command.Execute(context.Background()))

	result, fetchError := persistenceModule.Queries().ListStoresByChain(context.Background(), entities.StoreChainPAKNSAVE)
	require.NoError(testing, fetchError)
	assert.Len(testing, result, 1)
}

func TestSyncRetailersCommand_HandlesRetailerFailure(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	woolworthsStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainWoolworths, Id: "2", Name: "Woolworths Glenfield"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{error: fmt.Errorf("network timeout")},
		responses.StoreChainWoolworths: &mockRetailer{stores: woolworthsStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	baseCommand := aplication.BaseCommand{
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
