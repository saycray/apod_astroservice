package service

import (
	"astro"
	"context"
	"time"

	"astro/pkg/repository"

	_ "github.com/jmoiron/sqlx"
)

type AstroService struct {
	repo repository.Picture
}

func NewAstroService(repo repository.Picture) *AstroService {
	return &AstroService{repo}
}

func (s *AstroService) InsertOne(ctx context.Context, pic *astro.AstroModel) (int64, error) {
	return s.repo.InsertOne(ctx, pic)
}

func (s *AstroService) GetByDate(ctx context.Context, date time.Time) (*astro.AstroModel, error) {
	return s.repo.GetByDate(ctx, date)
}

func (s *AstroService) GetByDateRange(ctx context.Context, start, end time.Time) ([]astro.AstroModel, error) {
	return s.repo.GetByDateRange(ctx, start, end)
}

func (s *AstroService) DeleteByDate(ctx context.Context, date time.Time) (int64, error) {
	return s.repo.DeleteByDate(ctx, date)
}
