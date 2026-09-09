package presentation

import (
	"onion.api/aplication"
	"onion.api/presentation/controllers"

	"github.com/labstack/echo/v4"
)

type PresentationSettings struct {
	Port string
}

type PresentationModule struct {
	ServerInstance *echo.Echo
}

func Initialize(settings PresentationSettings, applicationModule *aplication.ApplicationModule) *PresentationModule {
	serverInstance := echo.New()
	serverInstance.HideBanner = true

	controllers.Initialize(serverInstance)
	controllers.InitializeRetailerController(serverInstance, applicationModule.SyncRetailersCommand)

	return &PresentationModule{
		ServerInstance: serverInstance,
	}
}

func (module *PresentationModule) Start(settings PresentationSettings) error {
	return module.ServerInstance.Start(settings.Port)
}
