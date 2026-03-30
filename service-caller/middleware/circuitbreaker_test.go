package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/sony/gobreaker"
)

func TestCircuitBreakerInit(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             "test-breaker",
	})

	cb := GetCircuitBreaker()
	if cb == nil {
		t.Error("Expected circuit breaker to be initialized")
	}
}

func TestCircuitBreakerExecute(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             "test-breaker-execute",
	})

	result, err := CircuitBreakerExecute(func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got '%s'", result)
	}
}

func TestCircuitBreakerStateTransition(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-state-transition",
		MaxRequests: 2,
		Interval:    0,
		Timeout:     1 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})

	_, _ = cb.Execute(func() (interface{}, error) {
		return "fail1", gobreaker.ErrOpenState
	})

	_, _ = cb.Execute(func() (interface{}, error) {
		return "fail2", gobreaker.ErrOpenState
	})

	if cb.State() != gobreaker.StateOpen {
		t.Errorf("Expected state Open, got %v", cb.State())
	}
}

func TestCircuitBreakerClosedAfterSuccess(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-closed-after-success",
		MaxRequests: 2,
		Interval:    0,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})

	for i := 0; i < 5; i++ {
		cb.Execute(func() (interface{}, error) {
			return "success", nil
		})
	}

	if cb.State() != gobreaker.StateClosed {
		t.Errorf("Expected state Closed after successes, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-half-open",
		MaxRequests: 2,
		Interval:    0,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})

	_, _ = cb.Execute(func() (interface{}, error) {
		return nil, gobreaker.ErrOpenState
	})
	_, _ = cb.Execute(func() (interface{}, error) {
		return nil, gobreaker.ErrOpenState
	})

	time.Sleep(150 * time.Millisecond)

	_, _ = cb.Execute(func() (interface{}, error) {
		return "half-open", nil
	})

	if cb.State() != gobreaker.StateHalfOpen {
		t.Errorf("Expected state HalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreakerHTTPExecute(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             "test-http-breaker",
	})

	resp, body, err := CircuitBreakerHTTPExecute(func() (*http.Response, []byte, error) {
		return nil, []byte("test body"), nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if string(body) != "test body" {
		t.Errorf("Expected 'test body', got '%s'", string(body))
	}
	if resp != nil {
		t.Error("Expected nil response")
	}
}

func TestRecordCircuitBreakerState(t *testing.T) {
	RecordCircuitBreakerState("test-breaker", 0)
	RecordCircuitBreakerState("test-breaker", 1)
	RecordCircuitBreakerState("test-breaker", 2)
}
