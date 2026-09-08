package woolworths

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"
	woolworthsresponses "onion.api/infrastrucre/woolworths/responses"
)

var cdxUrl = "https://api.cdx.nz/site-location/api/v1/sites"

type WoolworthsClient struct {
	*common.BaseRetailer
	storeName string
}

func NewWoolworthsClient(storeName string, degreeOfParallelism, numOfRetries, delayInMs, delayMaxInMs, timeoutInMs int, httpClient *http.Client, logger common.Logger) *WoolworthsClient {
	retailer := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "https://www.woolworths.co.nz",
		DegreeOfParallelism: degreeOfParallelism,
		NumOfRetries:        numOfRetries,
		DelayInMs:           delayInMs,
		DelayMaxInMs:        delayMaxInMs,
		Timeout:             time.Duration(timeoutInMs) * time.Millisecond,
		DefaultHeaders: map[string][]string{
			"User-Agent": {common.DefaultUserAgent},
		},
		Session: common.NoOpSession{},
		Name:    "woolworths",
		Logger:  logger,
	}, httpClient)

	return &WoolworthsClient{BaseRetailer: retailer, storeName: storeName}
}

func (client *WoolworthsClient) GetStores(requestContext context.Context) ([]responses.StoreResponse, error) {
	results, executeError := client.Execute(requestContext, []common.RetailClientRequest{
		{
			Method: "GET",
			Path:   cdxUrl,
			Label:  "woolworths sites",
			Headers: map[string][]string{
				"x-requested-with": {""},
				"Accept":           {"application/json, text/plain, */*"},
				"Origin":           {"https://www.woolworths.co.nz"},
				"Referer":          {"https://www.woolworths.co.nz/"},
			},
		},
	})
	if executeError != nil {
		return nil, responses.NewScrapingExceptionResponse("woolworths: fetch sites", executeError)
	}
	if !results[0].IsOk() {
		return nil, responses.NewScrapingExceptionResponse("woolworths: fetch sites", results[0].Error)
	}
	var siteResponse woolworthsresponses.SiteResponse
	if unmarshalError := json.Unmarshal(results[0].Response.Body, &siteResponse); unmarshalError != nil {
		return nil, responses.NewScrapingExceptionResponse("woolworths: decode sites", unmarshalError)
	}
	var stores []responses.StoreResponse
	for _, detail := range siteResponse.SiteDetail {
		address := detail.Site.AddressLine1
		if detail.Site.Suburb != "" {
			if address != "" {
				address += ", "
			}
			address += detail.Site.Suburb
		}
		if detail.Site.Postcode != "" {
			if address != "" {
				address += ", "
			}
			address += detail.Site.Postcode
		}
		stores = append(stores, responses.StoreResponse{
			Retailer:  responses.StoreChainWoolworths,
			Id:        fmt.Sprintf("%d", detail.Site.ID),
			Name:      detail.Site.Name,
			Address:   address,
			Latitude:  detail.Site.Latitude,
			Longitude: detail.Site.Longitude,
		})
	}
	if len(stores) == 0 {
		return nil, responses.NewScrapingExceptionResponse("woolworths: no stores returned", nil)
	}
	return stores, nil
}

func (client *WoolworthsClient) GetProducts(requestContext context.Context) ([]responses.ScrapedProductResponse, error) {
	return nil, nil
}
