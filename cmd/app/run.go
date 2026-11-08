package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func (a *App) Run() error {
	if a == nil || a.server == nil {
		return errors.New("aplicacion no inicializada")
	}
	serverErr := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(quit)
	select {
	case err := <-serverErr:
		return err
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()
		return a.Shutdown(ctx)
	}
}
