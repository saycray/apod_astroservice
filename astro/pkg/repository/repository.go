package repository

import (
	"astro"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type Picture interface {
	InsertOne(ctx context.Context, p *astro.AstroModel) (int64, error)
	GetByDate(ctx context.Context, date time.Time) (*astro.AstroModel, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]astro.AstroModel, error)
	DeleteByDate(ctx context.Context, date time.Time) (int64, error)
}

type Repository struct {
	Picture
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		Picture: NewPostgres(db),
	}
}
