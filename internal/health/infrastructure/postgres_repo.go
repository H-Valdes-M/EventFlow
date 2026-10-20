package infrastructure

import (
	"context"

	"eventflow/pkg/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresHealthRepository struct {
	db database.Database
}

func NewPostgresHealthRepository(db database.Database) *PostgresHealthRepository {
	return &PostgresHealthRepository{db: db}
}

// Implementamos la interfaz del repository
func (r *PostgresHealthRepository) Ping(ctx context.Context) error {
	pool := r.db.Native().(*pgxpool.Pool)
	return pool.Ping(ctx)
}
