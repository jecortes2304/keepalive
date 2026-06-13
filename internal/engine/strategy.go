package engine

import "context"

type MovementStrategy interface {
	Execute(ctx context.Context) error
	Name() string
}
