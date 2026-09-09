package retailers_test

import (
	"context"
	"fmt"
	"testing"

	retaileractions "onion.api/aplication/actions/retailers"
	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"

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

func TestFetchAllRetailerStores_ReturnsStoresFromAllRetailers(testing *testing.T) {
	paknsaveStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainPakNSave, Id: "1", Name: "Pak'nSave Mt Wellington"},
	}
	woolworthsStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainWoolworths, Id: "2", Name: "Woolworths Glenfield"},
		{Retailer: responses.StoreChainWoolworths, Id: "3", Name: "Woolworths Albany"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{stores: paknsaveStores},
		responses.StoreChainWoolworths: &mockRetailer{stores: woolworthsStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	stores, error := action.FetchAllRetailerStores(context.Background())

	require.NoError(testing, error)
	assert.Len(testing, stores, 3)
}

func TestFetchAllRetailerStores_ToleratesPartialFailure(testing *testing.T) {
	woolworthsStores := []responses.StoreResponse{
		{Retailer: responses.StoreChainWoolworths, Id: "2", Name: "Woolworths Glenfield"},
	}

	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{error: fmt.Errorf("network timeout")},
		responses.StoreChainWoolworths: &mockRetailer{stores: woolworthsStores},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	stores, error := action.FetchAllRetailerStores(context.Background())

	require.NoError(testing, error)
	assert.Len(testing, stores, 1)
	assert.Equal(testing, responses.StoreChainWoolworths, stores[0].Retailer)
}

func TestFetchAllRetailerStores_ReturnsErrorWhenAllFail(testing *testing.T) {
	retailers := map[responses.StoreChain]common.IRetailer{
		responses.StoreChainPakNSave:   &mockRetailer{error: fmt.Errorf("paknsave down")},
		responses.StoreChainWoolworths: &mockRetailer{error: fmt.Errorf("woolworths down")},
	}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	_, error := action.FetchAllRetailerStores(context.Background())

	assert.Error(testing, error)
}

func TestFetchAllRetailerStores_EmptyRetailers(testing *testing.T) {
	retailers := map[responses.StoreChain]common.IRetailer{}

	action := retaileractions.NewGetRetailersStoresAction(retailers)
	stores, error := action.FetchAllRetailerStores(context.Background())

	require.NoError(testing, error)
	assert.Empty(testing, stores)
}
