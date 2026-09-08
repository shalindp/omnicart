package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"onion.api/persistence/entities"
)

type PersistenceSettings struct {
	ConnectionString string
}

type PersistenceModule struct {
	pool    *pgxpool.Pool
	queries *entities.Queries
}

func Initialize(settings PersistenceSettings) (*PersistenceModule, error) {
	connectionPool, error := pgxpool.New(context.Background(), settings.ConnectionString)
	if error != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", error)
	}

	error = connectionPool.Ping(context.Background())
	if error != nil {
		return nil, fmt.Errorf("failed to ping database: %w", error)
	}

	queriesInstance := entities.New(connectionPool)

	return &PersistenceModule{
		pool:    connectionPool,
		queries: queriesInstance,
	}, nil
}

func (module *PersistenceModule) Queries() *entities.Queries {
	return module.queries
}

func (module *PersistenceModule) Pool() *pgxpool.Pool {
	return module.pool
}

func (module *PersistenceModule) Close() {
	module.pool.Close()
}
