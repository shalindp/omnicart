package infrastrucre

import (
	"fmt"
	"net/http"
	"time"

	"onion.api/infrastrucre/common"
	"onion.api/infrastrucre/paknsave"
	"onion.api/infrastrucre/woolworths"
	"onion.api/persistence"
)

type RetailerSettings struct {
	ReferenceStore      string
	DegreeOfParallelism int
	NumOfRetries        int
	DelayInMs           int
	DelayMaxInMs        int
	TimeoutInMs         int
}

type InfrastructureSettings struct {
	PakNSave   RetailerSettings
	Woolworths RetailerSettings
}

type InfrastructureModule struct {
	PakNSaveClient   *paknsave.PakNSaveClient
	WoolworthsClient *woolworths.WoolworthsClient
}

func Initialize(settings InfrastructureSettings, persistenceModule *persistence.PersistenceModule) (*InfrastructureModule, error) {
	logger := &stdoutLogger{}

	paknsaveHttpClient := &http.Client{Timeout: 30 * time.Second}
	paknsaveRetailer := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "https://api-prod.paknsave.co.nz/v1/edge",
		DegreeOfParallelism: settings.PakNSave.DegreeOfParallelism,
		NumOfRetries:        settings.PakNSave.NumOfRetries,
		DelayInMs:           settings.PakNSave.DelayInMs,
		DelayMaxInMs:        settings.PakNSave.DelayMaxInMs,
		Timeout:             time.Duration(settings.PakNSave.TimeoutInMs) * time.Millisecond,
		PersistenceModule:   persistenceModule,
		Name:                "paknsave",
		Logger:              logger,
	}, paknsaveHttpClient)
	paknsaveClient := paknsave.NewPakNSaveClient(paknsaveRetailer, settings.PakNSave.ReferenceStore)

	woolworthsHttpClient := &http.Client{Timeout: 30 * time.Second}
	woolworthsRetailer := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "https://www.woolworths.co.nz",
		DegreeOfParallelism: settings.Woolworths.DegreeOfParallelism,
		NumOfRetries:        settings.Woolworths.NumOfRetries,
		DelayInMs:           settings.Woolworths.DelayInMs,
		DelayMaxInMs:        settings.Woolworths.DelayMaxInMs,
		Timeout:             time.Duration(settings.Woolworths.TimeoutInMs) * time.Millisecond,
		DefaultHeaders: map[string][]string{
			"User-Agent": {common.DefaultUserAgent},
		},
		PersistenceModule: persistenceModule,
		Name:              "woolworths",
		Logger:            logger,
	}, woolworthsHttpClient)
	woolworthsClient := woolworths.NewWoolworthsClient(woolworthsRetailer, settings.Woolworths.ReferenceStore)

	return &InfrastructureModule{
		PakNSaveClient:   paknsaveClient,
		WoolworthsClient: woolworthsClient,
	}, nil
}

type stdoutLogger struct{}

func (logger *stdoutLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (logger *stdoutLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func (logger *stdoutLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (logger *stdoutLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}
