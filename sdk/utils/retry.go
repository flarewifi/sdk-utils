package sdkutils

import (
	"fmt"
	"time"
)

// Retry executes the given function up to 'retries' times.
// It waits 2s, 4s, 6s... between retries when errors occur.
// The function 'fn' should return (T, error).
func Retry[T any](fn func() (T, error), retries int) (T, error) {
	var result T
	var err error

	for attempt := 1; attempt <= retries; attempt++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}

		if attempt < retries {
			sleepDuration := time.Duration(attempt*2) * time.Second
			fmt.Printf("Attempt %d failed: %v. Retrying in %v...\n", attempt, err, sleepDuration)
			time.Sleep(sleepDuration)
		}
	}

	return result, fmt.Errorf("after %d retries, last error: %w", retries, err)
}
