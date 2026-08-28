package application

import (
	"context"

	"github.com/eternal-yellowduck/kanbandemo/internal/domain"
)

type Store interface {
	SaveTask(context.Context, domain.Task) error
	GetTask(context.Context, string) (domain.Task, bool, error)
	SaveRun(context.Context, domain.Run) error
	GetRun(context.Context, string) (domain.Run, bool, error)
	SaveAttempt(context.Context, domain.Attempt) error
	GetAttempt(context.Context, string) (domain.Attempt, bool, error)
	SaveCheck(context.Context, domain.Check) error
	GetCheck(context.Context, string) (domain.Check, bool, error)
	SaveBlock(context.Context, domain.Block) error
	GetBlock(context.Context, string) (domain.Block, bool, error)
	AppendEvents(context.Context, ...domain.DomainEvent) error
	ListEvents(context.Context, string) ([]domain.DomainEvent, error)
}
