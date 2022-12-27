package repository

import (
	"astro"
	"astro/pkg/consts"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	queryInsert = `INSERT INTO pictures
				   ("date", title, url, hd_url, thumbnail_url, media_type, copyright, explanation) 
				   VALUES($1, $2, $3, $4, $5, $6, $7, $8)`

	queryGetByDate = `SELECT "date", title, url, hd_url, thumbnail_url, media_type, copyright, explanation
					  FROM pictures WHERE "date" = $1`

	queryGetByDateRange = `SELECT "date", title, url, hd_url, thumbnail_url, media_type, copyright, explanation
					 	   FROM pictures WHERE "date" >= $1 AND "date" <= $2`

	queryDeleteByDate = `DELETE FROM pictures WHERE "date" = $1`
)

type Actions struct {
	db *sqlx.DB
}

func NewPostgres(db *sqlx.DB) *Actions {
	return &Actions{db}
}

func (r *Actions) InsertOne(ctx context.Context, d *astro.AstroModel) (int64, error) {

	result, err := r.db.ExecContext(ctx, queryInsert, time.Now(), d.Title, d.URL, d.HDURL, d.ThumbURL, d.MediaType, d.Copyright, d.Explanation)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()

}

func (r *Actions) GetByDate(ctx context.Context, date time.Time) (*astro.AstroModel, error) {

	var picture []astro.AstroModel
	if err := r.db.SelectContext(ctx, &picture, queryGetByDate, date.Format(consts.TimeFormat)); err != nil {
		return nil, err
	}

	if picture == nil {
		return nil, nil
	}

	return &picture[0], nil
}

func (r *Actions) GetByDateRange(ctx context.Context, start, end time.Time) ([]astro.AstroModel, error) {

	var pictures []astro.AstroModel
	if err := r.db.SelectContext(ctx, &pictures, queryGetByDateRange, start.Format(consts.TimeFormat), end.Format(consts.TimeFormat)); err != nil {
		return nil, err
	}

	return pictures, nil
}

func (r *Actions) DeleteByDate(ctx context.Context, date time.Time) (int64, error) {

	res, err := r.db.ExecContext(ctx, queryDeleteByDate, date.Format(consts.TimeFormat))
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
