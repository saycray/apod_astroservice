package service

import (
	"astro/pkg/repository"
	"context"
	"time"

	"astro"

	_ "github.com/jmoiron/sqlx"
)

type Picture interface {
	InsertOne(ctx context.Context, p *astro.AstroModel) (int64, error)
	GetByDate(ctx context.Context, date time.Time) (*astro.AstroModel, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]astro.AstroModel, error)
	DeleteByDate(ctx context.Context, date time.Time) (int64, error)
}

type Service struct {
	Picture
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Picture: NewAstroService(repos.Picture),
	}
}
