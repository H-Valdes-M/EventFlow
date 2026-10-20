package app

import (
	"context"
	"errors"
	"os"

	"eventflow/cmd/config"
	"eventflow/pkg/database"
	"eventflow/pkg/logger"

	"go.uber.org/zap"
)

// El relay y el consumidor se conectaran aqui cuando exista orders.
type WorkerResources struct {
	Config   *config.Configuration
	Logger   *zap.Logger
	Database database.Database
}

func InitWorker() (*WorkerResources, error) {
	cfg, err := config.LoadVars()
	if err != nil {
		return nil, err
	}
	log, err := logger.NewWithWriter(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	if err != nil {
		return nil, err
	}
	db, err := database.NewPostgresDatabase(context.Background(), database.PostgresConfig{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser, Password: cfg.DBPassword,
		Database: cfg.DBName, SSLMode: cfg.DBSSLMode,
	})
	if err != nil {
		_ = log.Sync()
		return nil, err
	}
	return &WorkerResources{Config: cfg, Logger: log, Database: db}, nil
}

// Run mantiene el proceso vivo y deja el punto de integración preparado para
// el relay del outbox y el consumidor RabbitMQ.
func (w *WorkerResources) Run(ctx context.Context) error {
	if w == nil || w.Config == nil || w.Database == nil {
		return errors.New("worker no inicializado")
	}
	<-ctx.Done()
	return nil
}

func (w *WorkerResources) Shutdown() error {
	if w == nil {
		return nil
	}
	if w.Database != nil {
		w.Database.Close()
	}
	if w.Logger != nil {
		return w.Logger.Sync()
	}
	return nil
}
