package fixtures

import (
	"context"

	"gorm.io/gorm"
)

type Fixtures struct {
	ctx context.Context
	db  *gorm.DB
}

func New(ctx context.Context, db *gorm.DB) *Fixtures {
	return &Fixtures{
		ctx: ctx,
		db:  db,
	}
}
