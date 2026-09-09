package persistence_test

import (
	"testing"

	"onion.api/persistence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeWithInvalidConnectionStringReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "postgres://invalid:invalid@localhost:99999/nonexistent?sslmode=disable",
	}

	module, error := persistence.Initialize(settings)

	require.Error(testing, error)
	if module != nil {
		module.Close()
	}
}

func TestInitializeWithEmptyConnectionStringReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "",
	}

	module, error := persistence.Initialize(settings)

	require.Error(testing, error)
	if module != nil {
		module.Close()
	}
}

func TestInitializeWithUnreachableHostReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "postgres://admin:admin@localhost:1/nonexistent?sslmode=disable&connect_timeout=1",
	}

	module, error := persistence.Initialize(settings)

	require.Error(testing, error)
	if module != nil {
		module.Close()
	}
}

func TestInitializeWithInvalidHostReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "postgres://admin:admin@doesnotexist.example.com:5432/omnicart_db?sslmode=disable&connect_timeout=1",
	}

	module, error := persistence.Initialize(settings)

	require.Error(testing, error)
	if module != nil {
		module.Close()
	}
	assert.False(testing, error == nil)
}
