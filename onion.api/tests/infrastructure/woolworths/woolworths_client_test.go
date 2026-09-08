package woolworths_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"onion.api/infrastrucre/woolworths"
)

func TestWoolworthsClient_GetStores(testing *testing.T) {
	httpClient := &http.Client{Timeout: 30 * time.Second}

	client := woolworths.NewWoolworthsClient(
		"Glenfield",
		6, 3, 400, 600, 30000,
		httpClient, &testLogger{t: testing},
	)

	stores, error := client.GetStores(context.Background())
	if error != nil {
		testing.Fatalf("GetStores returned error: %v", error)
	}

	if len(stores) == 0 {
		testing.Fatal("GetStores returned no stores")
	}

	testing.Logf("Woolworths returned %d stores", len(stores))

	for index, store := range stores {
		if store.Retailer == "" {
			testing.Errorf("store[%d]: Retailer is empty", index)
		}
		if store.Id == "" {
			testing.Errorf("store[%d]: Id is empty", index)
		}
		if store.Name == "" {
			testing.Errorf("store[%d]: Name is empty", index)
		}
		if store.Address == "" {
			testing.Errorf("store[%d]: Address is empty", index)
		}
		if store.Latitude == 0 {
			testing.Errorf("store[%d]: Latitude is zero", index)
		}
		if store.Longitude == 0 {
			testing.Errorf("store[%d]: Longitude is zero", index)
		}
	}
}

type testLogger struct {
	t *testing.T
}

func (logger *testLogger) Infof(format string, args ...interface{})  { logger.t.Logf(format, args...) }
func (logger *testLogger) Warnf(format string, args ...interface{})  { logger.t.Logf(format, args...) }
func (logger *testLogger) Errorf(format string, args ...interface{}) { logger.t.Logf(format, args...) }
func (logger *testLogger) Debugf(format string, args ...interface{}) { logger.t.Logf(format, args...) }
