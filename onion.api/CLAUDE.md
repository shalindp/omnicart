# Omnicart Onion API

Go backend server for Omnicart, built with Onion Architecture.

## Project Structure

```
├── presentaion/          # Presentation layer (API, entrypoint)
│   ├── main.go           # Entrypoint — bootstraps all modules in sequence
│   ├── .env              # All env variables defined here
│   └── api/              # Controllers (Echo handlers)
├── aplication/           # Application layer (business logic orchestration)
│   ├── base.go           # ICommand, IQuery, BaseCommand, BaseQuery
│   ├── commands/         # Write operations (DB mutations)
│   ├── queries/          # Read operations (DB queries)
│   ├── actions/          # Reusable/shared logic (see rules below)
│   ├── mappers/          # Type conversion functions (barcode, store chain, etc.)
│   └── dtos/             # Data transfer objects
├── infrastrucre/         # Infrastructure layer (external services, integrations)
│   ├── common/
│   │   ├── retail_client.go  # BaseRetailer — HTTP client with retries, throttling, sessions + IRetailer interface
│   │   ├── session.go        # RetailSession interface + InMemorySessionStore
│   │   └── responses/        # Shared infrastructure types (BarcodeType, ScrapedProductResponse, etc.)
│   ├── paknsave/         # Pak'nSave integration (embeds *BaseRetailer)
│   └── woolworths/       # Woolworths integration (embeds *BaseRetailer)
├── persistence/          # Persistence layer (database access)
│   ├── entities/         # sqlc-generated query code
│   ├── sql_queries/      # Hand-written SQL (input to sqlc)
│   ├── migrations/       # Goose migration files
│   └── persistence_module.go  # Module bootstrap
├── tests/                # ALL tests live here (mirrors source structure)
└── utils/                # Shared utilities (env loading, etc.)
```

## Architecture Rules (Onion / Layered)

Dependency flow is strictly downward. A layer may only depend on layers above it.

```
┌─────────────────────────────────────┐
│            Presentation             │  ← entrypoint (main.go, controllers)
│         references Application      │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│            Application              │  ← business logic (commands, queries, actions)
│   references Infrastructure &       │
│         Persistence                 │
└──────┬──────────────┬───────────────┘
       │              │
┌──────▼─────┐  ┌─────▼───────────────┐
│   Infra-   │  │    Persistence      │  ← DB access (entities, sqlc)
│ structure  │  │  no internal deps   │
│ (external  │  │                     │
│ services)  │  │                     │
└──────┬─────┘  └─────────────────────┘
       │              ▲
       └──────────────┘
         references
         Persistence
```

- **Persistence** → no internal deps (uses only stdlib + DB drivers)
- **Infrastructure** → can use Persistence
- **Application** → can use Infrastructure and Persistence
- **Presentation** → can use Application (and everything below)

**Never** reverse these dependencies. A persistence package must never import from application or presentation.

Each layer has a **Module file** (e.g. `ApplicationModule.go`, `InfrastructureModule.go`) responsible for bootstrapping that layer's dependencies. In `presentaion/main.go`, modules are called in the correct sequence to wire up the entire application.

## Commands, Queries, Actions

### Commands
- Handle **writes** to the database (INSERT, UPDATE, DELETE).
- Every command **must** use one or more transactions (typically one).
- Commands are responsible for committing the transaction.

### Queries
- Handle **reads** from the database (SELECT).
- Do not modify data.

### Actions
- Used when command/query logic becomes too complex and needs to be broken down.
- Actions are also used for **sharing logic** across commands/queries.
- Actions **CAN** read from the database.
- Actions **CAN** mutate data in memory.
- Actions **CAN NEVER** commit to the database — that responsibility stays with the command.
- If an action modifies data, the calling command persists the changes.

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go |
| API Framework | Echo |
| Database | PostgreSQL |
| Query Builder | sqlc |
| Migrations | Goose |
| Architecture | Onion (layered) |

## Database

- SQL queries are defined in `persistence/sql_queries/` and compiled by sqlc.
- Entity models live in `persistence/entities/` (auto-generated).
- Migrations live in `persistence/migrations/` and are managed with Goose.
- sqlc config lives in `persistence/sqlc.yaml`.
- All env vars for DB connection are in `presentaion/.env`.

## Testing

- **HARD RULE: All tests live in the `tests/` directory.** Never place `_test.go` files alongside source files. No exceptions.
- Mirror the source directory structure inside `tests/` (e.g. `tests/persistence/`, `tests/aplication/commands/`).
- Test packages use the `_test` suffix (e.g. `package persistence_test`) and import the source package.
- **Always run tests with `-count=1`** to disable caching. Set `GOFLAGS=-count=1` in your shell or use `go test -count=1`.

## Naming Conventions

- **No short variable names.** Every variable must be descriptive and fully spelled out.
  - `t` → `testing`, `err` → `error`, `ctx` → `context`, `tx` → `transaction`
  - `q` → `queries`, `pm` → `persistenceModule`, `m` → `module`, `s` → `settings`
  - `poolCfg` → `poolConfig`, `pool` → `connectionPool`, `v` → `value`, `n` → `number`
- Exception: auto-generated code in `persistence/entities/` (sqlc output) uses its own conventions and must not be manually edited.
- Exception: Go method receivers may use short names where the type is clear from context (e.g. `module *PersistenceModule`).

### sqlc Query Naming

sqlc queries in `persistence/sql_queries/` follow a `{Action}{TableName}` pattern:

| Pattern | Example | Use |
|---------|---------|-----|
| `Insert{Table}` | `InsertProduct` | Single-row INSERT |
| `Update{Table}` | `UpdateProduct` | Single-row UPDATE by PK |
| `Fetch{Table}{Criteria}` | `FetchProductByBarcodeValue` | Single-row SELECT |
| `Fetch{Table}ByID` | `FetchStoreProductByRetailerID` | Lookup by unique key |
| `Delete{Table}{Criteria}` | `DeleteStoreProductUnseen` | DELETE with filter |
| `Count{Table}{Scope}` | `CountActiveProducts` | COUNT queries |
| `InsertOrUpdate{Table}` | `InsertOrUpdateStore` | UPSERT / ON CONFLICT |
| `Get{Description}` | `GetExistingBarcodeByExternalProductId` | Complex queries without a single-table action |

- Use `Fetch` not `Get` or `Query` for SELECT queries (unless complex/joined)
- Use `InsertOrUpdate` for upserts (ON CONFLICT)
- Prefix with `UpdateSyncRun` not `CommitSyncRun` — use domain verbs, not persistence verbs
- All names are PascalCase, no underscores

### Visibility by Default

- **Functions and methods are private (lowercase) by default.** Only expose what external packages need.
- Public functions require justification — prefer keeping logic internal.
- Constructors (`New...`) and interfaces are the common exceptions.
- Commands expose only `Execute` and the constructor; all helper methods are private.
- `BaseRetailer` exposes only `Execute`; `dispose`, `stats`, `resolveUrl` are private.

## Conventions

- Module bootstrap files are named `{Layer}Module.go` (e.g. `ApplicationModule.go`).
- Keep layer boundaries strict — if unsure whether something belongs in actions vs commands, prefer the simpler option and refactor later.
- Env variables are loaded from `presentaion/.env` only.

## BaseRetailer (HTTP Client)

The `BaseRetailer` in `infrastrucre/common/retail_client.go` is a single-pump, batched HTTP client per retailer. It queues requests and drains them in fixed-size batches with throttling, retries, and session management.

- Constructor: `NewBaseRetailer(config, httpClient)` returns `*BaseRetailer`
- Only public method: `Execute(ctx, requests)` — enqueues requests and waits for all results
- All internal methods (`dispose`, `stats`, `resolveUrl`, etc.) are private
- Holds `Logger` (public) and `Queries` (public) for use by embedded retailer clients
- Retailer clients (PakNSave, Woolworths) embed `*BaseRetailer` and call `client.Execute(...)` directly

## Retailer Client Pattern

Retailer clients embed `*common.BaseRetailer` via pointer embedding:

```go
type PakNSaveClient struct {
    *common.BaseRetailer
    storeName     string
    enrichBarcode bool
}
```

This gives each retailer access to:
- `client.Execute(...)` — HTTP execution with retries/throttling
- `client.Logger` — structured logging
- `client.Queries` — database access (set via `retailer.Queries = persistenceModule.Queries()`)

Constructor takes `*common.BaseRetailer` (not a copy) to avoid copying the mutex:
```go
func NewPakNSaveClient(retailer *common.BaseRetailer, storeName string, enrichBarcode bool) *PakNSaveClient {
    return &PakNSaveClient{BaseRetailer: retailer, storeName: storeName, enrichBarcode: enrichBarcode}
}
```

## IRetailer Interface

Defined in `infrastrucre/common/retail_client.go`:
```go
type IRetailer interface {
    GetProducts(context context.Context) ([]responses.ScrapedProductResponse, error)
}
```

## Response Types

All infrastructure response types live in `infrastrucre/common/responses/`:
- `ScrapedProductResponse` — normalized product from any retailer
- `ScrapeResultResponse` — partial scrape result with failure info
- `ScrapingExceptionResponse` — scraping error with context
- `BarcodeType` — barcode classification (Gtin, Plu, Internal, None)
- `BarcodeResult` — classified barcode value + type
- `StoreChain` — retailer chain enum

## Barcode Classification

The `Classify` function in `responses/barcode_response.go` handles barcode normalization:
1. Trims whitespace, rejects empty/non-numeric strings
2. Validates as GTIN (8, 12, 13, or 14 digits with check digit)
3. Validates as PLU (4 or 5 digits)
4. **Pads with leading zeros** to try GTIN lengths (8, 12, 13, 14) — e.g. `70177169916` → `0070177169916`
5. Falls back to `BarcodeTypeInternal` if nothing matches

## RetailerClient Flow

The `BaseRetailer.Execute()` method is a single-pump, batched HTTP client per retailer. It queues requests and drains them in fixed-size batches with throttling, retries, and session management.

```
                          Execute()
                               │
                    ┌──────────▼──────────┐
                    │  Wrap requests into  │
                    │  Job array (one Job  │
                    │  per request)        │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Spawn cancel watcher │
                    │  (on parent ctx done) │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │      enqueue()       │
                    │  append to queue     │
                    │  signal pump via     │
                    │  work channel        │
                    └──────────┬──────────┘
                               │
          ┌────────────────────▼────────────────────┐
          │             pumpLoop() goroutine          │
          │  (single long-lived goroutine)            │
          │  (panic recovery restarts pump)           │
          │                                          │
          │  ┌──────────────────────────────────┐   │
          │  │  pump():                          │   │
          │  │  loop: wait on work/done channel  │   │
          │  └──────────────┬───────────────────┘   │
          │                 │                        │
          │  ┌──────────────▼──────────────────┐    │
          │  │  pace() — sleep until nextBatchAt │    │
          │  └──────────────┬──────────────────┘    │
          │                 │                        │
          │  ┌──────────────▼──────────────────┐    │
          │  │  takeBatch() — pop up to         │    │
          │  │  DegreeOfParallelism from queue   │    │
          │  └──────────────┬──────────────────┘    │
          │                 │                        │
          │  ┌──────────────▼──────────────────┐    │
          │  │  sessionHeaders()                │    │
          │  │  ├─ session.Prime()              │    │
          │  │  ├─ session.Headers() ──── yes ──┤    │
          │  │  └─ session.MintRequest()        │    │
          │  │     attemptWithTimeout()         │    │
          │  │     session.TryAccept()          │    │
          │  │     ├─ yes: return headers       │    │
          │  │     └─ no: fail entire batch     │    │
          │  └──────────────┬──────────────────┘    │
          │                 │                        │
          │  ┌──────────────▼──────────────────┐    │
          │  │  runBatch()                      │    │
          │  │  ├─ fan-out: N goroutines        │    │
          │  │  │  each calls attemptWithTimeout│    │
          │  │  ├─ fan-in: waitGroup.Wait()     │    │
          │  │  │                               │    │
          │  │  └─ for each outcome:            │    │
          │  │     ├─ cancelled? → deliver error│    │
          │  │     ├─ succeeded? → deliver OK   │    │
          │  │     ├─ session expired?          │    │
          │  │     │  → invalidate, requeue     │    │
          │  │     ├─ retryable & not exhausted?│    │
          │  │     │  → compute backoff, requeue│    │
          │  │     └─ terminal failure          │    │
          │  │        → deliver error           │    │
          │  └──────────────┬──────────────────┘    │
          │                 │                        │
          │  ┌──────────────▼──────────────────┐    │
          │  │  set nextBatchAt = now + delay   │    │
          │  │  (longest of nextDelay, backoff) │    │
          │  └─────────────────────────────────┘    │
          └──────────────────────────────────────────┘
                               │
          ┌────────────────────▼────────────────────┐
         │         attemptWithTimeout()              │
          │                                          │
          │  resolveUrl(path)                        │
          │       │                                  │
          │  create http.Request with timeout         │
          │       │                                  │
          │  merge defaultHeaders + request.Headers   │
          │       │                                  │
          │  httpClient.Do(request)                   │
          │       │                                  │
          │  ├─ timeout? → RetailClientTimeoutError│
          │  ├─ network error? → pass through         │
          │  └─ success: read body, return response   │
          └───────────────────────────────────────────┘
```
