package paknsave_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"onion.api/infrastrucre/common"
	"onion.api/infrastrucre/paknsave"
	"onion.api/tests/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPakNSaveClient_GetStores(testing *testing.T) {
	persistenceModule := testhelpers.SetupTestDB(testing)
	httpClient := &http.Client{Timeout: 30 * time.Second}

	retailer := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "https://api-prod.paknsave.co.nz/v1/edge",
		DegreeOfParallelism: 6,
		NumOfRetries:        3,
		DelayInMs:           400,
		DelayMaxInMs:        600,
		Timeout:             30 * time.Second,
		PersistenceModule:   persistenceModule,
		Name:                "paknsave",
		Logger:              &testLogger{t: testing},
	}, httpClient)

	client := paknsave.NewPakNSaveClient(retailer, "Pak'nSave Hibiscus Coast")

	stores, error := client.GetStores(context.Background())
	require.NoError(testing, error)
	require.NotEmpty(testing, stores, "PakNSave returned no stores")
	testing.Logf("PakNSave returned %d stores", len(stores))

	for index, store := range stores {
		assert.NotEmpty(testing, store.Retailer, "store[%d]: Retailer is empty", index)
		assert.NotEmpty(testing, store.Id, "store[%d]: Id is empty", index)
		assert.NotEmpty(testing, store.Name, "store[%d]: Name is empty", index)
		assert.NotEmpty(testing, store.Address, "store[%d]: Address is empty", index)
		assert.NotZero(testing, store.Latitude, "store[%d]: Latitude is zero", index)
		assert.NotZero(testing, store.Longitude, "store[%d]: Longitude is zero", index)
	}
}

type testLogger struct {
	t *testing.T
}

func (logger *testLogger) Infof(format string, args ...interface{})  { logger.t.Logf(format, args...) }
func (logger *testLogger) Warnf(format string, args ...interface{})  { logger.t.Logf(format, args...) }
func (logger *testLogger) Errorf(format string, args ...interface{}) { logger.t.Logf(format, args...) }
func (logger *testLogger) Debugf(format string, args ...interface{}) { logger.t.Logf(format, args...) }
