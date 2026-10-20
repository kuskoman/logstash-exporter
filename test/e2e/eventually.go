package e2e

import (
	"testing"
	"time"
)

const (
	defaultWaitTimeout  = 15 * time.Second
	defaultPollInterval = 100 * time.Millisecond
)

// Eventually calls condition until it returns nil or the timeout expires.
// On timeout the test fails with the last error returned by condition.
func Eventually(t *testing.T, timeout time.Duration, condition func() error) {
	t.Helper()

	if err := poll(timeout, condition); err != nil {
		t.Fatalf("condition not met within %s: %v", timeout, err)
	}
}

// Consistently calls condition until the duration expires and fails the test
// as soon as it returns an error.
func Consistently(t *testing.T, duration time.Duration, condition func() error) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if err := condition(); err != nil {
			t.Fatalf("condition stopped holding: %v", err)
		}
		time.Sleep(defaultPollInterval)
	}
}

func poll(timeout time.Duration, condition func() error) error {
	deadline := time.Now().Add(timeout)
	for {
		err := condition()
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(defaultPollInterval)
	}
}
