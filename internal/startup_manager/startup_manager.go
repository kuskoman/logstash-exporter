package startup_manager

import (
	"context"
	"time"
)

const ServerShutdownTimeout = 10 * time.Second

// AppServer defines the behavior of an application server
type AppServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}
