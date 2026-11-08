package database

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresConfig contiene solo los datos necesarios para crear el pool. No se
// debe registrar ni devolver como parte de mensajes de error públicos.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// PostgresDatabase implementa Database sobre GORM con el driver PostgreSQL.
type PostgresDatabase struct {
	db *gorm.DB
}

// NewPostgresDatabase crea el cliente GORM sin afirmar conectividad. La disponibilidad
// se comprueba más tarde mediante Ping con el contexto y plazo del llamador.
func NewPostgresDatabase(ctx context.Context, config PostgresConfig) (*PostgresDatabase, error) {
	db, err := gorm.Open(postgres.Open(postgresDSN(config)), &gorm.Config{})
	if err != nil {
		return nil, errors.New("configuracion de PostgreSQL invalida")
	}
	return &PostgresDatabase{db: db.WithContext(ctx)}, nil
}

// Ping comprueba conectividad sin ocultar el contexto de cancelación del
// llamador.
func (d *PostgresDatabase) Ping(ctx context.Context) error {
	if d == nil || d.db == nil {
		return errors.New("provider PostgreSQL no inicializado")
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close libera el pool. Es seguro para instancias no inicializadas.
func (d *PostgresDatabase) Close() {
	if d == nil || d.db == nil {
		return
	}
	sqlDB, err := d.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

// Native expone el pool real solo para adaptadores de infraestructura.
func (d *PostgresDatabase) Native() any {
	if d == nil {
		return nil
	}
	return d.db
}

// AsGORM obtiene el cliente GORM o devuelve un error controlado para impedir
// assertions que puedan causar panic en infraestructura.
func AsGORM(database Database) (*gorm.DB, error) {
	if database == nil {
		return nil, errors.New("provider de base de datos no configurado")
	}
	db, ok := database.Native().(*gorm.DB)
	if !ok || db == nil {
		return nil, errors.New("provider de base de datos incompatible con PostgreSQL")
	}
	return db, nil
}

func postgresDSN(config PostgresConfig) string {
	connection := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Path:   config.Database,
	}
	query := connection.Query()
	query.Set("sslmode", config.SSLMode)
	connection.RawQuery = query.Encode()
	return connection.String()
}
