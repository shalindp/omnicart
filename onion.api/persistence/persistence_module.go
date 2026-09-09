package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (module *PersistenceModule) InTransaction(requestContext context.Context, transactionFunction func(queries *entities.Queries, transaction pgx.Tx) error) error {
	transaction, transactionError := module.pool.Begin(requestContext)
	if transactionError != nil {
		return fmt.Errorf("begin transaction: %w", transactionError)
	}

	transactionQueries := entities.New(transaction)
	functionError := transactionFunction(transactionQueries, transaction)

	if functionError != nil {
		_ = transaction.Rollback(requestContext)
		return functionError
	}

	if commitError := transaction.Commit(requestContext); commitError != nil {
		return fmt.Errorf("commit transaction: %w", commitError)
	}
	return nil
}

func (module *PersistenceModule) Close() {
	module.pool.Close()
}
