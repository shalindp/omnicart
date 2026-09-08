package presentation

import (
	"onion.api/infrastrucre"
	"onion.api/presentation/controllers"

	"github.com/labstack/echo/v4"
)

type PresentationSettings struct{}

type PresentationModule struct {
	ServerInstance *echo.Echo
}

func Initialize(settings PresentationSettings, infrastructureModule *infrastrucre.InfrastructureModule) *PresentationModule {
	serverInstance := echo.New()
	serverInstance.HideBanner = true

	controllers.Initialize(serverInstance)
	controllers.InitializeRetailerController(serverInstance, infrastructureModule.PakNSaveClient, infrastructureModule.WoolworthsClient)

	return &PresentationModule{
		ServerInstance: serverInstance,
	}
}

func (module *PresentationModule) Start(address string) error {
	return module.ServerInstance.Start(address)
}
