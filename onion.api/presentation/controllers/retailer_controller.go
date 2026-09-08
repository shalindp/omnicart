package controllers

import (
	"net/http"

	"onion.api/infrastrucre/common/responses"
	"onion.api/infrastrucre/paknsave"
	"onion.api/infrastrucre/woolworths"

	"github.com/labstack/echo/v4"
)

var pakNSaveClientRef *paknsave.PakNSaveClient
var woolworthsClientRef *woolworths.WoolworthsClient

func InitializeRetailerController(serverInstance *echo.Echo, pakNSaveClient *paknsave.PakNSaveClient, woolworthsClient *woolworths.WoolworthsClient) {
	pakNSaveClientRef = pakNSaveClient
	woolworthsClientRef = woolworthsClient
	serverInstance.GET("/retailers/stores", getStores)
}

func getStores(context echo.Context) error {
	requestContext := context.Request().Context()

	pakNSaveStores, pakNSaveError := pakNSaveClientRef.GetStores(requestContext)
	if pakNSaveError != nil {
		pakNSaveStores = nil
	}

	woolworthsStores, woolworthsError := woolworthsClientRef.GetStores(requestContext)
	if woolworthsError != nil {
		woolworthsStores = nil
	}

	allStores := make([]responses.StoreResponse, 0, len(pakNSaveStores)+len(woolworthsStores))
	allStores = append(allStores, pakNSaveStores...)
	allStores = append(allStores, woolworthsStores...)

	return context.JSON(http.StatusOK, allStores)
}
