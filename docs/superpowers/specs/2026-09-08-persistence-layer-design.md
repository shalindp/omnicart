# Persistence Layer — Cut 1 Design

## Goal

Port the .NET persistence layer to Go: database setup, sqlc codegen, migrations, environment configuration, and module bootstrap. This is the foundation that all subsequent layers build on.

## Scope

**Included:** All 10 tables, all enums, 9 SQL query files (~35 queries), all 14 migration files, environment variable loading, Makefile for migrations.

**Excluded:** `product_search.sql` (3 recursive CTE queries — next cut).

## Tables

| Table | Purpose |
|-------|---------|
| `product` | Cross-retailer product identity (barcode, name, brand, pack_size) |
| `store` | Retailer chain + region (WOOLWORTHS/PAKNSAVE/NEWWORLD + region_id) |
| `store_product` | Links product to store with retailer's external_product_id |
| `store_product_price` | Append-only price observations with sale_price_cents |
| `sync_run` | Tracks sync runs with heartbeat, status, stats |
| `product_image` | Multi-image support per product per chain |
| `retailer_session` | Per-chain API tokens |
| `category` | Adjacency-list category tree |
| `category_normaliser` | Maps retailer words to canonical category names |
| `product_category` | Links product to one category node |

## Enums

| Enum | Values |
|------|--------|
| `barcode_kind` | GTIN, PLU, INTERNAL |
| `store_chain` | WOOLWORTHS, PAKNSAVE, NEWWORLD |
| `sync_status` | RUNNING, SUCCEEDED, FAILED |

## File Structure

```
onion.api/
├── go.mod
├── Makefile                                    # Migration commands (up/down/reset/status)
├── persistence/
│   ├── sqlc.yaml                               # sqlc config for Go codegen
│   ├── migrations/                             # 14 .sql files copied from .NET as-is
│   │   ├── 00001_product.sql
│   │   ├── 00002_store.sql
│   │   ├── ... (14 total)
│   │   └── 00014_store_product_price_sale.sql
│   ├── sql_queries/                            # 9 .sql files adapted for Go sqlc
│   │   ├── product.sql
│   │   ├── store.sql
│   │   ├── store_product.sql
│   │   ├── sync_run.sql
│   │   ├── product_image.sql
│   │   ├── category.sql
│   │   ├── category_normaliser.sql
│   │   ├── retailer_session.sql
│   │   └── product_category.sql
│   ├── entities/                               # sqlc-generated (do not edit)
│   │   ├── db.go
│   │   ├── models.go
│   │   ├── product.sql.go
│   │   ├── store.sql.go
│   │   ├── store_product.sql.go
│   │   ├── sync_run.sql.go
│   │   ├── product_image.sql.go
│   │   ├── category.sql.go
│   │   ├── category_normaliser.sql.go
│   │   ├── retailer_session.sql.go
│   │   └── product_category.sql.go
│   └── persistence_module.go                   # PersistenceSettings + Initialize()
├── application/
│   └── application_module.go                   # Stub: ApplicationSettings + Initialize()
├── infrastrucre/
│   ├── common/                                 # Already exists (retail_client.go, etc.)
│   └── infrastructure_module.go                # Stub: InfrastructureSettings + Initialize()
├── presentation/
│   ├── .env                                    # DATABASE_URL, TEST_DATABASE_URL
│   ├── environment_variable_loader_util.go     # Load .env, Get/MustGet helpers
│   └── main.go                                 # Bootstrap: load env → settings → Initialize per layer
└── utils/
    └── env.go                                  # Low-level env loading (godotenv wrapper)
```

## Components

### 1. `go.mod`

Module path: `onion.api`

Dependencies:
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/pressly/goose/v3` — migrations (for Makefile only, not runtime)
- `github.com/sqlc-dev/sqlc` — codegen tool (not a runtime dep)
- `github.com/joho/godotenv` — .env file loading

### 2. `persistence/sqlc.yaml`

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

### 3. Migrations

Copy all 14 files from `/mnt/apps/__dev/CartMe/Backend/Persistence/Migrations/` into `persistence/migrations/` unchanged. They are already Goose format.

### 4. SQL Queries

Port 9 files from .NET `SqlQueries/` to Go sqlc format. Changes from .NET:
- Remove `csharp` plugin casts (`::barcode_kind`, `::store_chain` etc.) — Go sqlc infers types from schema
- Keep `sqlc.arg('name')` and `sqlc.narg('name')` syntax (same in Go sqlc)
- Keep `sqlc.arg('rows')::jsonb` for bulk operations (Go sqlc handles this)

### 5. `persistence/persistence_module.go`

```go
type PersistenceSettings struct {
    ConnectionString string
}

type PersistenceModule struct {
    db      *sql.DB
    queries *entities.Queries
}

func Initialize(settings PersistenceSettings) (*PersistenceModule, error)
func (module *PersistenceModule) Queries() *entities.Queries
func (module *PersistenceModule) DB() *sql.DB
```

Initialize opens the connection pool, pings to verify, returns module. No migrations.

### 6. Module Stubs

`application/application_module.go` and `infrastrucre/infrastructure_module.go` follow the same pattern:
- Settings struct with relevant env vars
- Initialize function (returns stub module for now)

### 7. `presentation/.env`

```
DATABASE_URL=postgres://admin:admin@localhost:5432/omnicart_db?sslmode=disable
TEST_DATABASE_URL=postgres://admin:admin@localhost:5432/omnicart_db_test?sslmode=disable
```

### 8. `presentation/environment_variable_loader_util.go`

Loads `presentation/.env` via godotenv. Exposes:
- `Load()` — reads .env, sets os env vars
- `Get(key string) string` — os.Getenv wrapper
- `MustGet(key string) string` — panics if missing

### 9. `presentation/main.go`

Bootstrap sequence:
1. Call env loader to load all vars
2. Create `persistence.PersistenceSettings` from env
3. Call `persistence.Initialize(settings)`
4. (Future: create settings for other layers, pass dependencies)

### 10. `Makefile`

```makefile
include presentation/.env
export

DB_URL := $(DATABASE_URL)

migrate-up:
    goose -dir persistence/migrations postgres $(DB_URL) up

migrate-down:
    goose -dir persistence/migrations postgres $(DB_URL) down

migrate-reset:
    goose -dir persistence/migrations postgres $(DB_URL) reset

migrate-status:
    goose -dir persistence/migrations postgres $(DB_URL) status
```

## Implementation Order

1. Create `go.mod` with dependencies
2. Copy 14 migration files into `persistence/migrations/`
3. Create `persistence/sqlc.yaml`
4. Port 9 SQL query files into `persistence/sql_queries/`
5. Run `sqlc generate` to produce `persistence/entities/`
6. Write `persistence/persistence_module.go`
7. Write stub module files (`application_module.go`, `infrastructure_module.go`)
8. Write `presentation/.env`
9. Write `presentation/environment_variable_loader_util.go`
10. Write `utils/env.go`
11. Write `presentation/main.go`
12. Write `Makefile`
13. Verify: `sqlc generate` succeeds, `go build ./...` compiles

## Verification

- `sqlc generate` produces entities without errors
- `go build ./...` compiles all packages
- `go vet ./...` passes
- Makefile targets work (requires running Postgres)
