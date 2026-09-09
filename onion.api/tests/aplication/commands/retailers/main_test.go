package retailers_test

import (
	"os"
	"testing"

	_ "github.com/lib/pq"
	"onion.api/tests/testhelpers"
)

func TestMain(m *testing.M) {
	testhelpers.StartPostgres()
	code := m.Run()
	testhelpers.ShutdownPostgres()
	os.Exit(code)
}
