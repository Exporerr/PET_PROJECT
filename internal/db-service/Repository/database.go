package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

// НОВЫЙ ПУЛ СОЕДИНЕНИЙ
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// НОВОЕ ХРАНИЛИЩЕ
func NewUserPool(pool *pgxpool.Pool) *Storage {
	return &Storage{db: pool}
}
