package controllers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func Initialize(serverInstance *echo.Echo) {
	serverInstance.GET("/_version", getVersion)
	serverInstance.GET("/_health", getHealth)
}

func getVersion(context echo.Context) error {
	return context.JSON(http.StatusOK, "1.0.0")
}

func getHealth(context echo.Context) error {
	currentTimeUTC := time.Now().UTC()
	newZealandLocation, locationError := time.LoadLocation("Pacific/Auckland")
	if locationError != nil {
		newZealandLocation = time.UTC
	}
	currentTimeNewZealand := currentTimeUTC.In(newZealandLocation)

	return context.JSON(http.StatusOK, map[string]string{
		"utc":     currentTimeUTC.Format(time.RFC3339),
		"nz":      currentTimeNewZealand.Format(time.RFC3339),
		"status":  "healthy",
	})
}
