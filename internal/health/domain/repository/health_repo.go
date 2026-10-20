package domain

import "context"

type HealthRepository interface {
	Ping(ctx context.Context) error
}
