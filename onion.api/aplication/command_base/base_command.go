package command_base

import (
	"context"
	"log"

	"onion.api/persistence"
)

type ICommand interface {
	Execute(requestContext context.Context) error
}

type IQuery interface {
	Execute(requestContext context.Context) error
}

type BaseCommand struct {
	PersistenceModule *persistence.PersistenceModule
	Logger            *log.Logger
}
