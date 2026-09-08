package infrastrucre

import (
	"fmt"
	"net/http"
	"time"

	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/paknsave"
	"onion.api/infrastrucre/woolworths"
)

type RetailerSettings struct {
	ReferenceStore       string
	DegreeOfParallelism  int
	NumOfRetries         int
	DelayInMs            int
	DelayMaxInMs         int
	TimeoutInMs          int
}

type InfrastructureSettings struct {
	PakNSave  RetailerSettings
	Woolworths RetailerSettings
}

type InfrastructureModule struct {
	PakNSaveClient   *paknsave.PakNSaveClient
	WoolworthsClient *woolworths.WoolworthsClient
}

func Initialize(settings InfrastructureSettings) (*InfrastructureModule, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	logger := &stdoutLogger{}

	pakNSaveRetailer := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "https://www.paknsave.co.nz",
		DegreeOfParallelism: settings.PakNSave.DegreeOfParallelism,
		NumOfRetries:        settings.PakNSave.NumOfRetries,
		DelayInMs:           settings.PakNSave.DelayInMs,
		DelayMaxInMs:        settings.PakNSave.DelayMaxInMs,
		Timeout:             time.Duration(settings.PakNSave.TimeoutInMs) * time.Millisecond,
		Name:                "paknsave",
		Logger:              logger,
	}, httpClient)

	woolworthsRetailer := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "https://www.woolworths.co.nz",
		DegreeOfParallelism: settings.Woolworths.DegreeOfParallelism,
		NumOfRetries:        settings.Woolworths.NumOfRetries,
		DelayInMs:           settings.Woolworths.DelayInMs,
		DelayMaxInMs:        settings.Woolworths.DelayMaxInMs,
		Timeout:             time.Duration(settings.Woolworths.TimeoutInMs) * time.Millisecond,
		Name:                "woolworths",
		Logger:              logger,
	}, httpClient)

	return &InfrastructureModule{
		PakNSaveClient:   paknsave.NewPakNSaveClient(pakNSaveRetailer, settings.PakNSave.ReferenceStore),
		WoolworthsClient: woolworths.NewWoolworthsClient(woolworthsRetailer),
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
