package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	envutil "eventflow/pkg/env"
)

// Configuration contiene la configuración compartida por API y worker.
// AMQP queda disponible para la futura integración del procesamiento de orders.
type Configuration struct {
	AppEnv                string
	Port                  int
	LogLevel              string
	LogFormat             string
	ShutdownTimeout       time.Duration
	RequestTimeout        time.Duration
	DBHost                string
	DBPort                int
	DBUser                string
	DBPassword            string
	DBName                string
	DBSSLMode             string
	AMQPURL               string
	OutboxPollInterval    time.Duration
	OutboxBatchSize       int
	PublishConfirmTimeout time.Duration
	ConsumerPrefetch      int
}

// LoadVars mantiene la forma de configuración del bootstrap del profesor,
// pero usa las dependencias y nombres propios de EventFlow.
func LoadVars() (*Configuration, error) {
	file, err := loadEnvironmentFile()
	if err != nil {
		return nil, err
	}

	port, err := readPort("PORT", 8080, file)
	if err != nil {
		return nil, err
	}
	dbPort, err := readPort("DB_PORT", 5432, file)
	if err != nil {
		return nil, err
	}
	requestTimeout, err := readDuration("REQUEST_TIMEOUT", 10*time.Second, file)
	if err != nil {
		return nil, err
	}
	shutdownTimeout, err := readDuration("SHUTDOWN_TIMEOUT", 15*time.Second, file)
	if err != nil {
		return nil, err
	}
	pollInterval, err := readDuration("OUTBOX_POLL_INTERVAL", 500*time.Millisecond, file)
	if err != nil {
		return nil, err
	}
	publishTimeout, err := readDuration("PUBLISH_CONFIRM_TIMEOUT", 5*time.Second, file)
	if err != nil {
		return nil, err
	}
	batchSize, err := readPositiveInt("OUTBOX_BATCH_SIZE", 20, file)
	if err != nil {
		return nil, err
	}
	prefetch, err := readPositiveInt("CONSUMER_PREFETCH", 1, file)
	if err != nil {
		return nil, err
	}

	cfg := &Configuration{
		AppEnv:                readString("ENVIRONMENT", "dev", file),
		Port:                  port,
		LogLevel:              readString("LOG_LEVEL", "info", file),
		LogFormat:             readString("LOG_FORMAT", "json", file),
		ShutdownTimeout:       shutdownTimeout,
		RequestTimeout:        requestTimeout,
		DBHost:                readString("DB_HOST", "localhost", file),
		DBPort:                dbPort,
		DBUser:                readString("DB_USER", "eventflow", file),
		DBPassword:            readString("DB_PASSWORD", "eventflow", file),
		DBName:                readString("DB_NAME", "eventflow", file),
		DBSSLMode:             readString("DB_SSL_MODE", "disable", file),
		AMQPURL:               readString("AMQP_URL", "amqp://guest:guest@localhost:5672/", file),
		OutboxPollInterval:    pollInterval,
		OutboxBatchSize:       batchSize,
		PublishConfirmTimeout: publishTimeout,
		ConsumerPrefetch:      prefetch,
	}
	if cfg.AppEnv != "dev" && cfg.AppEnv != "qa" && cfg.AppEnv != "prod" {
		return nil, fmt.Errorf("ENVIRONMENT: %q no es valido", cfg.AppEnv)
	}
	if cfg.LogLevel != "debug" && cfg.LogLevel != "info" && cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
		return nil, fmt.Errorf("LOG_LEVEL: %q no es valido", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" && cfg.LogFormat != "console" {
		return nil, fmt.Errorf("LOG_FORMAT: %q no es valido", cfg.LogFormat)
	}
	return cfg, nil
}

func loadEnvironmentFile() (map[string]string, error) {
	name := envutil.Normalize(os.Getenv("ENVIRONMENT"))
	if name == "prod" {
		return map[string]string{}, nil
	}
	return envutil.LoadDotEnv(".env." + name)
}

func lookup(key string, file map[string]string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return value, true
	}
	value, ok := file[key]
	return value, ok
}

func readString(key, fallback string, file map[string]string) string {
	if value, ok := lookup(key, file); ok {
		return value
	}
	return fallback
}

func readPort(key string, fallback int, file map[string]string) (int, error) {
	value, err := strconv.Atoi(readString(key, strconv.Itoa(fallback), file))
	if err != nil || value < 1 || value > 65535 {
		return 0, fmt.Errorf("%s: puerto invalido", key)
	}
	return value, nil
}

func readDuration(key string, fallback time.Duration, file map[string]string) (time.Duration, error) {
	value, err := time.ParseDuration(readString(key, fallback.String(), file))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s: duracion invalida", key)
	}
	return value, nil
}

func readPositiveInt(key string, fallback int, file map[string]string) (int, error) {
	value, err := strconv.Atoi(readString(key, strconv.Itoa(fallback), file))
	if err != nil || value < 1 {
		return 0, fmt.Errorf("%s: se espera un entero positivo", key)
	}
	return value, nil
}
