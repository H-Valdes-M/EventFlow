package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"eventflow/cmd/config"
	"eventflow/cmd/container"
	"eventflow/cmd/routes"
	"eventflow/pkg/database"
	"eventflow/pkg/logger"
	"eventflow/pkg/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "eventflow/doc"
)

type App struct {
	config    *config.Configuration
	container *container.Container
	router    *gin.Engine
	log       *zap.Logger
	db        database.Database
	server    *http.Server
}

func NewApp() *App { return &App{} }

func (a *App) Init() error {
	cfg, err := config.LoadVars()
	if err != nil {
		return err
	}
	log, err := logger.NewWithWriter(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	if err != nil {
		return err
	}
	db, err := database.NewPostgresDatabase(context.Background(), database.PostgresConfig{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser, Password: cfg.DBPassword,
		Database: cfg.DBName, SSLMode: cfg.DBSSLMode,
	})
	if err != nil {
		_ = log.Sync()
		return err
	}
	c, err := container.NewContainer(db, log)
	if err != nil {
		db.Close()
		_ = log.Sync()
		return err
	}

	router := gin.New()

	// Aplicamos Middlewares
	router.Use(gin.Logger())

	router.Use(middleware.ErrorHandlerMiddleware(a.log)) // Global Error Handler
	// Middleware Helmet
	router.Use(middleware.SecurityHeaders())
	// Middleware Cors
	router.Use(cors.Default())

	if cfg.AppEnv != "prod" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	apiRouter := routes.NewAPIRouter(c)
	apiRouter.RegisterRoutes(router)

	a.config, a.log, a.db = cfg, log, db
	a.container, a.router = c, router
	a.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.RequestTimeout,
		WriteTimeout:      cfg.RequestTimeout,
		IdleTimeout:       60 * time.Second,
	}
	return nil
}
