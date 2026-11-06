package app

import (
	"context"
	"errors"
	"os"

	"eventflow/cmd/config"
	orderinfra "eventflow/internal/order/infrastructure/postgres"
	"eventflow/pkg/database"
	"eventflow/pkg/logger"
	"eventflow/pkg/pubsub"

	"go.uber.org/zap"
)

// El relay y el consumidor se conectaran aqui cuando exista orders.
type WorkerResources struct {
	Config   *config.Configuration
	Logger   *zap.Logger
	Database database.Database
	RabbitMQ *pubsub.RabbitMQProvider
	Relay    *orderinfra.OutboxRelay
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
	rabbit, err := pubsub.NewRabbitMQProvider(context.Background(), cfg.AMQPURL)
	if err != nil {
		db.Close()
		_ = log.Sync()
		return nil, err
	}
	relay, err := orderinfra.NewOutboxRelay(db, rabbit)
	if err != nil {
		rabbit.Close()
		db.Close()
		_ = log.Sync()
		return nil, err
	}
	return &WorkerResources{Config: cfg, Logger: log, Database: db, RabbitMQ: rabbit, Relay: relay}, nil
}

// Run publica los eventos pendientes del outbox en RabbitMQ.
func (w *WorkerResources) Run(ctx context.Context) error {
	if w == nil || w.Config == nil || w.Database == nil || w.Relay == nil {
		return errors.New("worker no inicializado")
	}
	return w.Relay.Run(ctx, w.Config.OutboxPollInterval, w.Config.OutboxBatchSize, w.Config.PublishConfirmTimeout)
}

func (w *WorkerResources) Shutdown() error {
	if w == nil {
		return nil
	}
	if w.RabbitMQ != nil {
		w.RabbitMQ.Close()
	}
	if w.Database != nil {
		w.Database.Close()
	}
	if w.Logger != nil {
		return w.Logger.Sync()
	}
	return nil
}
