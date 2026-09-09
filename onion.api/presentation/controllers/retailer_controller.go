package controllers

import (
	"net/http"

	retailercommands "onion.api/aplication/commands/retailers"

	"github.com/labstack/echo/v4"
)

func InitializeRetailerController(serverInstance *echo.Echo, syncCommand *retailercommands.SyncRetailersCommand) {
	serverInstance.POST("/retailers/sync", syncRetailers(syncCommand))
}

func syncRetailers(command *retailercommands.SyncRetailersCommand) echo.HandlerFunc {
	return func(context echo.Context) error {
		executeError := command.Execute(context.Request().Context())
		if executeError != nil {
			return context.JSON(http.StatusInternalServerError, map[string]string{"error": executeError.Error()})
		}
		return context.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
