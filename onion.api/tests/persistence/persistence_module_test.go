package persistence_test

import (
	"testing"

	"onion.api/persistence"
)

func TestInitializeWithInvalidConnectionStringReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "postgres://invalid:invalid@localhost:99999/nonexistent?sslmode=disable",
	}

	module, error := persistence.Initialize(settings)

	if error == nil {
		module.Close()
		testing.Fatal("expected error for invalid connection string, got nil")
	}
}

func TestInitializeWithEmptyConnectionStringReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "",
	}

	module, error := persistence.Initialize(settings)

	if error == nil {
		module.Close()
		testing.Fatal("expected error for empty connection string, got nil")
	}
}

func TestInitializeWithUnreachableHostReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "postgres://admin:admin@localhost:1/nonexistent?sslmode=disable&connect_timeout=1",
	}

	module, error := persistence.Initialize(settings)

	if error == nil {
		module.Close()
		testing.Fatal("expected error for unreachable host, got nil")
	}
}

func TestInitializeWithInvalidHostReturnsError(testing *testing.T) {
	settings := persistence.PersistenceSettings{
		ConnectionString: "postgres://admin:admin@doesnotexist.example.com:5432/omnicart_db?sslmode=disable&connect_timeout=1",
	}

	module, error := persistence.Initialize(settings)

	if error == nil {
		module.Close()
		testing.Fatal("expected error for invalid host, got nil")
	}
}
