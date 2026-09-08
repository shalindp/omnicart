package infrastrucre

import (
	"fmt"
	"net/http"
	"time"

	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/paknsave"
	paknsaveresponses "onion.api/infrastrucre/paknsave/responses"
	"onion.api/infrastrucre/woolworths"
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

func Initialize(settings InfrastructureSettings) (*InfrastructureModule, error) {
	logger := &stdoutLogger{}

	paknsaveSessionStore := common.NewInMemorySessionStore()
	paknsaveSession := paknsaveresponses.NewPakNSaveSessionProvider(paknsaveSessionStore)

	paknsaveHttpClient := &http.Client{Timeout: 30 * time.Second}
	paknsaveClient := paknsave.NewPakNSaveClient(
		settings.PakNSave.ReferenceStore,
		settings.PakNSave.DegreeOfParallelism,
		settings.PakNSave.NumOfRetries,
		settings.PakNSave.DelayInMs,
		settings.PakNSave.DelayMaxInMs,
		settings.PakNSave.TimeoutInMs,
		paknsaveHttpClient, paknsaveSession, logger,
	)

	woolworthsHttpClient := &http.Client{Timeout: 30 * time.Second}
	woolworthsClient := woolworths.NewWoolworthsClient(
		settings.Woolworths.ReferenceStore,
		settings.Woolworths.DegreeOfParallelism,
		settings.Woolworths.NumOfRetries,
		settings.Woolworths.DelayInMs,
		settings.Woolworths.DelayMaxInMs,
		settings.Woolworths.TimeoutInMs,
		woolworthsHttpClient, logger,
	)

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
