# Persistence Layer — Cut 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the .NET persistence layer to Go — database setup, sqlc codegen, migrations, environment configuration, and module bootstrap.

**Architecture:** sqlc generates Go code from hand-written SQL queries + Goose migration schemas. Each layer has a module file with a settings struct and `Initialize` function. `main.go` loads env vars, creates settings, and calls `Initialize` per layer bottom-up.

**Tech Stack:** Go, PostgreSQL (pgx/v5), sqlc, Goose (migrations via Makefile), godotenv

**Spec:** `docs/superpowers/specs/2026-09-08-persistence-layer-design.md`

## Global Constraints

- Module path: `onion.api`
- Directory name `infrastrucre/` is intentional (matches existing code)
- All tests live in `tests/` directory (mirrors source structure)
- Variable names must be descriptive (no short names like `t`, `err`, `ctx`)
- Functions private by default; only expose what external packages need
- sqlc queries use `{Action}{TableName}` naming pattern (Fetch, Insert, Update, etc.)

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `go.mod` | Create | Module definition + dependencies |
| `persistence/migrations/*.sql` | Copy | 14 migration files from .NET |
| `persistence/sqlc.yaml` | Create | sqlc config for Go codegen |
| `persistence/sql_queries/product.sql` | Create | Product queries |
| `persistence/sql_queries/store.sql` | Create | Store queries |
| `persistence/sql_queries/store_product.sql` | Create | Store product queries |
| `persistence/sql_queries/sync_run.sql` | Create | Sync run queries |
| `persistence/sql_queries/product_image.sql` | Create | Product image queries |
| `persistence/sql_queries/category.sql` | Create | Category queries |
| `persistence/sql_queries/category_normaliser.sql` | Create | Category normaliser queries |
| `persistence/sql_queries/retailer_session.sql` | Create | Retailer session queries |
| `persistence/sql_queries/product_category.sql` | Create | Product category queries |
| `persistence/entities/` | Generated | sqlc output (do not edit) |
| `persistence/persistence_module.go` | Create | PersistenceSettings + Initialize() |
| `application/application_module.go` | Create | Stub module |
| `infrastrucre/infrastructure_module.go` | Create | Stub module |
| `presentation/.env` | Create | DATABASE_URL, TEST_DATABASE_URL |
| `utils/env.go` | Create | Low-level env loading |
| `presentation/environment_variable_loader_util.go` | Create | Load .env, Get/MustGet |
| `presentation/main.go` | Create | Bootstrap entrypoint |
| `Makefile` | Create | Migration commands |

---

### Task 1: Create go.mod and install dependencies

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

```bash
cd onion.api
go mod init onion.api
```

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/jackc/pgx/v5
go get github.com/pressly/goose/v3
go get github.com/joho/godotenv
go get github.com/lib/pq
```

- [ ] **Step 3: Verify**

Run: `cat go.mod`
Expected: module `onion.api` with all four dependencies listed

---

### Task 2: Copy migration files

**Files:**
- Create: `persistence/migrations/00001_product.sql` through `00014_store_product_price_sale.sql`

- [ ] **Step 1: Create migrations directory and copy all 14 files**

```bash
mkdir -p persistence/migrations
cp /mnt/apps/__dev/CartMe/Backend/Persistence/Migrations/*.sql persistence/migrations/
```

- [ ] **Step 2: Verify count**

Run: `ls persistence/migrations/ | wc -l`
Expected: 14

---

### Task 3: Create sqlc.yaml

**Files:**
- Create: `persistence/sqlc.yaml`

- [ ] **Step 1: Write sqlc config**

```yaml
version: "2"

sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "sql_queries"
    codegen:
      - out: "entities"
        package: "entities"
        engine: "postgresql"
        sql_package: "pgx/v5"
```

- [ ] **Step 2: Verify**

Run: `cd persistence && sqlc version`
Expected: sqlc is installed and reports version

---

### Task 4: Port SQL query files

**Files:**
- Create: `persistence/sql_queries/product.sql`
- Create: `persistence/sql_queries/store.sql`
- Create: `persistence/sql_queries/store_product.sql`
- Create: `persistence/sql_queries/sync_run.sql`
- Create: `persistence/sql_queries/product_image.sql`
- Create: `persistence/sql_queries/category.sql`
- Create: `persistence/sql_queries/category_normaliser.sql`
- Create: `persistence/sql_queries/retailer_session.sql`
- Create: `persistence/sql_queries/product_category.sql`

**Source:** `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/`

**Adaptation rules for each file:**
- Keep all SQL exactly as-is
- Keep all `-- name: ... :one/:many/:exec/:execrows` annotations exactly as-is
- Keep all `sqlc.arg('name')` and `sqlc.narg('name')` syntax exactly as-is
- Keep all `sqlc.arg('rows')::jsonb` casts for bulk operations
- Remove any C#-specific comments (lines mentioning "sqlc-gen-csharp", "C#", "Persistence/Queries")
- Keep all other comments (they document the business logic)

- [ ] **Step 1: Copy and adapt product.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/product.sql`, remove C#-specific comments, write to `persistence/sql_queries/product.sql`

- [ ] **Step 2: Copy and adapt store.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/store.sql`, remove C#-specific comments, write to `persistence/sql_queries/store.sql`

- [ ] **Step 3: Copy and adapt store_product.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/store_product.sql`, remove C#-specific comments, write to `persistence/sql_queries/store_product.sql`

- [ ] **Step 4: Copy and adapt sync_run.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/sync_run.sql`, remove C#-specific comments, write to `persistence/sql_queries/sync_run.sql`

- [ ] **Step 5: Copy and adapt product_image.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/product_image.sql`, remove C#-specific comments, write to `persistence/sql_queries/product_image.sql`

- [ ] **Step 6: Copy and adapt category.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/category.sql`, remove C#-specific comments, write to `persistence/sql_queries/category.sql`

- [ ] **Step 7: Copy and adapt category_normaliser.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/category_normaliser.sql`, remove C#-specific comments, write to `persistence/sql_queries/category_normaliser.sql`

- [ ] **Step 8: Copy and adapt retailer_session.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/retailer_session.sql`, remove C#-specific comments, write to `persistence/sql_queries/retailer_session.sql`

- [ ] **Step 9: Copy and adapt product_category.sql**

Read `/mnt/apps/__dev/CartMe/Backend/Persistence/SqlQueries/product_category.sql`, remove C#-specific comments, write to `persistence/sql_queries/product_category.sql`

- [ ] **Step 10: Verify file count**

Run: `ls persistence/sql_queries/ | wc -l`
Expected: 9

---

### Task 5: Run sqlc generate

**Files:**
- Generated: `persistence/entities/` (all .go files)

- [ ] **Step 1: Run sqlc generate**

```bash
cd persistence && sqlc generate
```

- [ ] **Step 2: Verify generated files**

Run: `ls persistence/entities/`
Expected: `db.go`, `models.go`, `querier.go`, plus one `.sql.go` file per query file

- [ ] **Step 3: Verify models have correct types**

Check that `persistence/entities/models.go` contains structs for all 10 tables with correct field types (uuid.UUID for PKs, sql.NullString for nullable text, etc.)

---

### Task 6: Write persistence_module.go

**Files:**
- Create: `persistence/persistence_module.go`

- [ ] **Step 1: Write the module file**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"onion.api/persistence/entities"
)

type PersistenceSettings struct {
	ConnectionString string
}

type PersistenceModule struct {
	db      *sql.DB
	queries *entities.Queries
}

func Initialize(settings PersistenceSettings) (*PersistenceModule, error) {
	databaseConnection, error := sql.Open("postgres", settings.ConnectionString)
	if error != nil {
		return nil, fmt.Errorf("failed to open database: %w", error)
	}

	databaseConnection.SetMaxOpenConns(25)
	databaseConnection.SetMaxIdleConns(5)
	databaseConnection.SetConnMaxLifetime(5 * time.Minute)

	pingContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	error = databaseConnection.PingContext(pingContext)
	if error != nil {
		return nil, fmt.Errorf("failed to ping database: %w", error)
	}

	queriesInstance := entities.New(databaseConnection)

	return &PersistenceModule{
		db:      databaseConnection,
		queries: queriesInstance,
	}, nil
}

func (module *PersistenceModule) Queries() *entities.Queries {
	return module.queries
}

func (module *PersistenceModule) DB() *sql.DB {
	return module.db
}

func (module *PersistenceModule) Close() error {
	return module.db.Close()
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./persistence/...`
Expected: compiles without errors

---

### Task 7: Write stub module files

**Files:**
- Create: `application/application_module.go`
- Create: `infrastrucre/infrastructure_module.go`

- [ ] **Step 1: Write application module stub**

```go
package application

type ApplicationSettings struct {
	PakNSaveEnrichBarcodes bool
}

type ApplicationModule struct{}

func Initialize(settings ApplicationSettings) (*ApplicationModule, error) {
	return &ApplicationModule{}, nil
}
```

- [ ] **Step 2: Write infrastructure module stub**

```go
package infrastrucre

type InfrastructureSettings struct {
	WoolworthsReferenceStore  string
	PakNSaveReferenceStore    string
}

type InfrastructureModule struct{}

func Initialize(settings InfrastructureSettings) (*InfrastructureModule, error) {
	return &InfrastructureModule{}, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./application/... && go build ./infrastrucre/...`
Expected: both compile without errors

---

### Task 8: Write .env and environment variable loader

**Files:**
- Create: `presentation/.env`
- Create: `utils/env.go`
- Create: `presentation/environment_variable_loader_util.go`

- [ ] **Step 1: Create .env file**

```
DATABASE_URL=postgres://admin:admin@localhost:5432/omnicart_db?sslmode=disable
TEST_DATABASE_URL=postgres://admin:admin@localhost:5432/omnicart_db_test?sslmode=disable
```

- [ ] **Step 2: Write utils/env.go**

```go
package utils

import (
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv(filePath string) error {
	return godotenv.Load(filePath)
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
```

- [ ] **Step 3: Write presentation/environment_variable_loader_util.go**

```go
package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"onion.api/utils"
)

func loadEnvironmentVariables() {
	_, currentFile, _, runtimeReference := runtime.Caller(0)
	presentationDirectory := filepath.Dir(currentFile)
	envFilePath := filepath.Join(presentationDirectory, ".env")

	error := utils.LoadEnv(envFilePath)
	if error != nil {
		fmt.Printf("Warning: could not load .env file: %v\n", error)
	}
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./utils/... && go build ./presentation/...`
Expected: compiles without errors (main.go doesn't exist yet, so presentation will fail — that's expected)

---

### Task 9: Write main.go

**Files:**
- Create: `presentation/main.go`

- [ ] **Step 1: Write main.go**

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"onion.api/persistence"
)

func main() {
	loadEnvironmentVariables()

	databaseConnectionString := os.Getenv("DATABASE_URL")
	if databaseConnectionString == "" {
		fmt.Println("Error: DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	persistenceSettings := persistence.PersistenceSettings{
		ConnectionString: databaseConnectionString,
	}

	persistenceModule, error := persistence.Initialize(persistenceSettings)
	if error != nil {
		fmt.Printf("Error initializing persistence layer: %v\n", error)
		os.Exit(1)
	}
	defer persistenceModule.Close()

	fmt.Println("Application started successfully")

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	fmt.Println("Shutting down...")
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./presentation/...`
Expected: compiles without errors

---

### Task 10: Write Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Write Makefile**

```makefile
include presentation/.env
export

DB_URL := $(DATABASE_URL)

.PHONY: migrate-up migrate-down migrate-reset migrate-status sqlc-generate

migrate-up:
	goose -dir persistence/migrations postgres $(DB_URL) up

migrate-down:
	goose -dir persistence/migrations postgres $(DB_URL) down

migrate-reset:
	goose -dir persistence/migrations postgres $(DB_URL) reset

migrate-status:
	goose -dir persistence/migrations postgres $(DB_URL) status

sqlc-generate:
	cd persistence && sqlc generate
```

- [ ] **Step 2: Verify Makefile syntax**

Run: `make --dry-run migrate-status`
Expected: shows the goose command that would run (won't execute without Postgres)

---

### Task 11: Final verification

**Files:**
- None (verification only)

- [ ] **Step 1: Run go vet**

```bash
go vet ./...
```

Expected: no issues

- [ ] **Step 2: Run go build**

```bash
go build ./...
```

Expected: all packages compile

- [ ] **Step 3: Verify sqlc generate still works**

```bash
cd persistence && sqlc generate
```

Expected: no errors, entities regenerated

- [ ] **Step 4: Commit all files**

```bash
git add -A
git commit -m "feat: persistence layer — sqlc, migrations, module bootstrap, env setup"
```
