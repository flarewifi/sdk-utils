package sdkutils

import (
	"context"
	"time"
)

func SleepContext(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		// The context was cancelled
		return ctx.Err()
	case <-time.After(delay):
		// The delay duration elapsed
		return nil
	}
}
