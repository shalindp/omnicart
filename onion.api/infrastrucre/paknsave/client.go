package paknsave

import (
	"context"

	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"
)

type PakNSaveClient struct {
	*common.BaseRetailer
	storeName string
}

func NewPakNSaveClient(retailer *common.BaseRetailer, storeName string) *PakNSaveClient {
	return &PakNSaveClient{
		BaseRetailer: retailer,
		storeName:    storeName,
	}
}

func (client *PakNSaveClient) GetStores(requestContext context.Context) ([]responses.StoreResponse, error) {
	results, executeError := client.Execute(requestContext, []common.RetailClientRequest{
		{Method: "GET", Path: ApiBaseUrl + "/store", Label: "paknsave stores"},
	})
	if executeError != nil {
		return nil, responses.NewScrapingExceptionResponse("paknsave: fetch stores", executeError)
	}
	if !results[0].IsOk() {
		return nil, responses.NewScrapingExceptionResponse("paknsave: fetch stores", results[0].Error)
	}
	storeList, deserializeError := DeserializeStoreList(results[0].Response.Body, "paknsave: decode stores")
	if deserializeError != nil {
		return nil, deserializeError
	}
	var stores []responses.StoreResponse
	for _, store := range storeList.Stores {
		stores = append(stores, responses.StoreResponse{
			Retailer:  responses.StoreChainPakNSave,
			Id:        store.Id,
			Name:      store.Name,
			Address:   store.Address,
			Latitude:  store.Latitude,
			Longitude: store.Longitude,
		})
	}
	return stores, nil
}

func (client *PakNSaveClient) GetProducts(requestContext context.Context) ([]responses.ScrapedProductResponse, error) {
	return nil, nil
}
