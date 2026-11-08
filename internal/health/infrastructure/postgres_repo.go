package infrastructure

import (
	"context"

	"eventflow/pkg/database"
)

type PostgresHealthRepository struct {
	db database.Database
}

func NewPostgresHealthRepository(db database.Database) *PostgresHealthRepository {
	return &PostgresHealthRepository{db: db}
}

// Implementamos la interfaz del repository
func (r *PostgresHealthRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
