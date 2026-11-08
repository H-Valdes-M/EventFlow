package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"eventflow/cmd/config"
	"eventflow/internal/order/application/consumer"
	"eventflow/internal/order/domain/entity"
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
	Consumer *consumer.Handler
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
	relay, err := orderinfra.NewOutboxRelay(db, rabbit, log)
	if err != nil {
		rabbit.Close()
		db.Close()
		_ = log.Sync()
		return nil, err
	}
	store, err := orderinfra.NewOrderConsumerStore(db)
	if err != nil {
		rabbit.Close()
		db.Close()
		_ = log.Sync()
		return nil, err
	}
	return &WorkerResources{Config: cfg, Logger: log, Database: db, RabbitMQ: rabbit, Relay: relay, Consumer: consumer.New(store)}, nil
}

// Run coordina el relay y el consumidor hasta que se solicite el apagado.
func (w *WorkerResources) Run(ctx context.Context) error {
	if w == nil || w.Config == nil || w.Database == nil || w.Relay == nil || w.RabbitMQ == nil || w.Consumer == nil {
		return errors.New("worker no inicializado")
	}
	log := w.Logger
	if log == nil {
		log = zap.NewNop()
	}
	log.Info("Worker iniciado", zap.String("consumer", consumer.Name))
	intake, stop := context.WithCancel(ctx)
	defer stop()
	processing, cancelProcessing := context.WithCancel(context.Background())
	defer cancelProcessing()
	results := make(chan error, 2)
	go func() {
		results <- w.Relay.Run(intake, w.Config.OutboxPollInterval, w.Config.OutboxBatchSize, w.Config.PublishConfirmTimeout)
	}()
	go func() {
		results <- w.RabbitMQ.Consume(intake, processing, w.Config.ConsumerPrefetch, logDelivery(log, w.Consumer.Handle))
	}()
	var first error
	received := false
	select {
	case <-ctx.Done():
	case first = <-results:
		received = true
		stop()
	}
	stop()
	log.Info("Worker deteniendo recepcion de eventos")
	deadline, cancel := context.WithTimeout(context.Background(), w.Config.ShutdownTimeout)
	defer cancel()
	remaining := 2
	if received {
		remaining--
	}
	for remaining > 0 {
		select {
		case err := <-results:
			remaining--
			if first == nil && err != nil {
				first = err
			}
		case <-deadline.Done():
			cancelProcessing()
			log.Error("Tiempo de apagado del worker agotado")
			return deadline.Err()
		}
	}
	if first != nil {
		log.Error("Worker detenido por un error")
	} else {
		log.Info("Worker detenido")
	}
	return first
}

// logDelivery registra la decisión del consumidor, no la confirmación AMQP.
func logDelivery(log *zap.Logger, handle pubsub.DeliveryHandler) pubsub.DeliveryHandler {
	return func(ctx context.Context, delivery pubsub.Delivery) (pubsub.Disposition, error) {
		disposition, err := handle(ctx, delivery)
		if err != nil {
			log.Error("No se pudo completar la entrega; queda sin confirmar")
			return disposition, err
		}
		if disposition == pubsub.NackDrop {
			log.Warn("Evento invalido registrado para rechazo")
			return disposition, nil
		}
		if disposition == pubsub.Ack {
			var event entity.OrderCreatedEvent
			if json.Unmarshal(delivery.Payload, &event) == nil {
				log.Info("Resultado de entrega registrado; confirmacion pendiente",
					zap.String("event_id", event.EventID), zap.String("order_id", event.OrderID),
					zap.String("message_id", delivery.MessageID), zap.String("event_type", event.EventType))
			}
		}
		return disposition, nil
	}
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
