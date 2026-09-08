package presentation

import (
	"onion.api/presentation/controllers"

	"github.com/labstack/echo/v4"
)

type PresentationSettings struct{}

type PresentationModule struct {
	ServerInstance *echo.Echo
}

func Initialize(settings PresentationSettings) *PresentationModule {
	serverInstance := echo.New()
	serverInstance.HideBanner = true

	controllers.Initialize(serverInstance)

	return &PresentationModule{
		ServerInstance: serverInstance,
	}
}

func (module *PresentationModule) Start(address string) error {
	return module.ServerInstance.Start(address)
}
