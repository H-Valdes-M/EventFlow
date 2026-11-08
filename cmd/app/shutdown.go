package app

import "context"

func (a *App) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var firstErr error
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			firstErr = err
			_ = a.server.Close()
		}
	}
	if a.db != nil {
		a.db.Close()
	}
	if a.log != nil {
		if err := a.log.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
