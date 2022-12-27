package repository

import (
	"context"
	"fmt"
	"time"

	_ "database/sql"

	"github.com/jmoiron/sqlx"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	SSLMode  string
}

func NewPostgresDB(ctx context.Context, c Config) (*sqlx.DB, error) {

	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", c.Host, c.Port, c.Username, c.DBName, c.Password, c.SSLMode)
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(10 * time.Second)
	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(5)

	return db, nil
}
