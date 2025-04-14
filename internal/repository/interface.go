package repository

import (
	"context"
	"runbin/internal/model"
)

type PasteRepository interface {
	Save(p *model.Paste) error
	Update(p *model.Paste) error
	GetByID(id string) (*model.Paste, bool)
	DispatchExecutionTask(id string) error
	GetTask(ctx context.Context) (*model.Paste, error)
}

