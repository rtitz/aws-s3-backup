package utils

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// Retry configuration constants
const (
	MaxRetryDuration = 12 * time.Hour
	InitialDelay     = 1 * time.Second
	MaxDelay         = 5 * time.Minute
)

// RetryWithBackoff retries a function with exponential backoff for network errors
func RetryWithBackoff(ctx context.Context, operation func() error, operationName string) error {
	startTime := time.Now()
	attempt := 0
	delay := InitialDelay

	for {
		err := operation()
		if err == nil {
			logSuccessAfterRetries(operationName, attempt)
			return nil
		}

		// If not a network error, fail immediately
		if !IsNetworkError(err) {
			return err
		}

		attempt++
		elapsed := time.Since(startTime)

		// Check if we've exceeded max retry duration
		if elapsed >= MaxRetryDuration {
			return fmt.Errorf("❌ %s failed after %v of retries: %w", operationName, MaxRetryDuration, err)
		}

		// Calculate next delay with exponential backoff
		nextDelay := calculateNextDelay(delay, attempt)
		
		logRetryAttempt(operationName, attempt, err, nextDelay)

		// Wait with context cancellation support
		if err := waitWithContext(ctx, nextDelay); err != nil {
			return err
		}

		delay = nextDelay
	}
}

// IsNetworkError checks if an error is network-related
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	networkErrors := []string{
		"network", "timeout", "connection", "dial", "dns",
		"temporary", "i/o timeout", "no route to host",
		"connection refused", "connection reset",
	}

	for _, networkErr := range networkErrors {
		if strings.Contains(errStr, networkErr) {
			return true
		}
	}
	return false
}

// logSuccessAfterRetries logs success message after retry attempts
func logSuccessAfterRetries(operationName string, attempt int) {
	if attempt > 0 {
		log.Printf("✅ %s succeeded after %d attempts", operationName, attempt+1)
	}
}

// calculateNextDelay computes exponential backoff delay
func calculateNextDelay(currentDelay time.Duration, attempt int) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * math.Pow(2, float64(attempt-1)))
	if nextDelay > MaxDelay {
		return MaxDelay
	}
	return nextDelay
}

// logRetryAttempt logs retry attempt with error and next delay
func logRetryAttempt(operationName string, attempt int, err error, nextDelay time.Duration) {
	log.Printf("⚠️ %s failed (attempt %d): %v", operationName, attempt, err)
	log.Printf("🔄 Retrying in %v... (Press Ctrl+C to cancel, will retry for up to 12 hours)", nextDelay)
}

// waitWithContext waits for delay duration with context cancellation support
func waitWithContext(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}