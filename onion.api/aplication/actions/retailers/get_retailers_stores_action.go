package retailers

import (
	"context"
	"fmt"
	"sync"

	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"
)

type GetRetailersStoresAction struct {
	Retailers map[responses.StoreChain]common.IRetailer
}

func NewGetRetailersStoresAction(retailers map[responses.StoreChain]common.IRetailer) *GetRetailersStoresAction {
	return &GetRetailersStoresAction{Retailers: retailers}
}

func (action *GetRetailersStoresAction) FetchAllRetailerStores(requestContext context.Context) ([]responses.StoreResponse, error) {
	type chainResult struct {
		stores []responses.StoreResponse
		error  error
	}
	resultChannel := make(chan chainResult, len(action.Retailers))

	var waitGroup sync.WaitGroup
	for chain, retailer := range action.Retailers {
		waitGroup.Add(1)
		go func(chain responses.StoreChain, retailer common.IRetailer) {
			defer waitGroup.Done()
			stores, fetchError := retailer.GetStores(requestContext)
			resultChannel <- chainResult{stores: stores, error: fetchError}
		}(chain, retailer)
	}
	waitGroup.Wait()
	close(resultChannel)

	var allStores []responses.StoreResponse
	var firstError error
	for result := range resultChannel {
		if result.error != nil {
			if firstError == nil {
				firstError = fmt.Errorf("get retailers stores: %w", result.error)
			}
			continue
		}
		allStores = append(allStores, result.stores...)
	}
	if firstError != nil && len(allStores) == 0 {
		return nil, firstError
	}
	return allStores, nil
}
