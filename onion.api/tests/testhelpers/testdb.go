package testhelpers

import (
	"os"
	"path/filepath"
	"testing"

	"blake.io/pqx/pqxtest"
	"github.com/pressly/goose/v3"
	"onion.api/persistence"
)

func init() {
	goose.SetDialect("postgres")
}

// SetupTestDB creates an isolated test database with all migrations applied
// and returns a wired PersistenceModule. Each test gets its own database
// that is automatically cleaned up when the test finishes.
//
// The calling test package must have a TestMain that starts Postgres:
//
//	func TestMain(m *testing.M) {
//	    testhelpers.StartPostgres()
//	    code := m.Run()
//	    testhelpers.ShutdownPostgres()
//	    os.Exit(code)
//	}
func SetupTestDB(testing *testing.T) *persistence.PersistenceModule {
	testing.Helper()

	database := pqxtest.CreateDB(testing, "")
	migrationsDirectory := migrationsPath(testing)

	gooseError := goose.Up(database, migrationsDirectory)
	if gooseError != nil {
		testing.Fatalf("setup test db: run migrations: %v", gooseError)
	}

	dsn := pqxtest.DSNForTest(testing)
	persistenceModule, initializeError := persistence.Initialize(persistence.PersistenceSettings{
		ConnectionString: dsn,
	})
	if initializeError != nil {
		testing.Fatalf("setup test db: initialize persistence: %v", initializeError)
	}

	testing.Cleanup(func() {
		persistenceModule.Close()
	})

	return persistenceModule
}

// StartPostgres starts the embedded Postgres instance. Call from TestMain.
func StartPostgres() {
	pqxtest.Start(30e9, 0)
}

// ShutdownPostgres shuts down the embedded Postgres instance. Call from TestMain.
func ShutdownPostgres() {
	pqxtest.Shutdown()
}

func migrationsPath(testing *testing.T) string {
	testing.Helper()

	currentDirectory, error := os.Getwd()
	if error != nil {
		testing.Fatalf("find project root: %v", error)
	}

	for {
		goModPath := filepath.Join(currentDirectory, "go.mod")
		if _, statError := os.Stat(goModPath); statError == nil {
			return filepath.Join(currentDirectory, "persistence", "migrations")
		}

		parent := filepath.Dir(currentDirectory)
		if parent == currentDirectory {
			testing.Fatalf("find project root: go.mod not found")
		}
		currentDirectory = parent
	}
}
