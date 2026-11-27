package cli

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestExecuteWithTimeoutReportsStall proves a blocked command surfaces as an
// error instead of hanging the process.
func TestExecuteWithTimeoutReportsStall(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	start := time.Now()
	err := executeWithTimeout(func() error {
		<-block
		return nil
	}, 50*time.Millisecond, "focus add")

	if err == nil {
		t.Fatal("expected a stall error, got nil")
	}
	if !strings.Contains(err.Error(), "did not finish within") {
		t.Fatalf("unexpected error text: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("stall error took %s, expected the 50ms bound to fire first", elapsed)
	}
}

// TestExecuteWithTimeoutReturnsCommandError keeps ordinary command failures
// untouched.
func TestExecuteWithTimeoutReturnsCommandError(t *testing.T) {
	want := errors.New("boom")
	if got := executeWithTimeout(func() error { return want }, time.Second, "focus list"); !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// TestCommandTimeout covers the default and the environment override.
func TestCommandTimeout(t *testing.T) {
	t.Setenv(commandTimeoutEnv, "")
	if got := commandTimeout(); got != defaultCommandTimeout {
		t.Fatalf("default: expected %s, got %s", defaultCommandTimeout, got)
	}

	t.Setenv(commandTimeoutEnv, "1500ms")
	if got := commandTimeout(); got != 1500*time.Millisecond {
		t.Fatalf("override: expected 1.5s, got %s", got)
	}

	t.Setenv(commandTimeoutEnv, "not-a-duration")
	if got := commandTimeout(); got != defaultCommandTimeout {
		t.Fatalf("invalid override should fall back to %s, got %s", defaultCommandTimeout, got)
	}
}
