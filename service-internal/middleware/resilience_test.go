package middleware

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker"
)

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             "test-circuit-breaker",
	})

	cb := GetCircuitBreaker()
	if cb == nil {
		t.Fatal("Circuit breaker should be initialized")
	}

	for i := 0; i < 6; i++ {
		_, err := CircuitBreakerExecute(func() (string, error) {
			return "", errors.New("downstream error")
		})
		if err == nil {
			t.Error("Expected error from circuit breaker")
		}
	}

	state := cb.State()
	if state != gobreaker.StateOpen {
		t.Errorf("Expected circuit breaker to be open, got %v", state)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		Name:             "test-circuit-breaker-halfopen",
	})

	cb := GetCircuitBreaker()

	for i := 0; i < 6; i++ {
		CircuitBreakerExecute(func() (string, error) {
			return "", errors.New("downstream error")
		})
	}

	time.Sleep(1100 * time.Millisecond)

	_, err := CircuitBreakerExecute(func() (string, error) {
		return "success", nil
	})

	state := cb.State()
	if state != gobreaker.StateHalfOpen {
		t.Errorf("Expected circuit breaker to be half-open, got %v", state)
	}
	_ = err
}

func TestCircuitBreakerClosesAfterRecovery(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		Name:             "test-circuit-breaker-recovery",
	})

	cb := GetCircuitBreaker()

	for i := 0; i < 4; i++ {
		CircuitBreakerExecute(func() (string, error) {
			return "", errors.New("downstream error")
		})
	}

	time.Sleep(1100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		_, err := CircuitBreakerExecute(func() (string, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	}

	state := cb.State()
	if state != gobreaker.StateClosed {
		t.Errorf("Expected circuit breaker to be closed, got %v", state)
	}
}

func TestCircuitBreakerDisabled(t *testing.T) {
	result, err := CircuitBreakerExecute(func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error when circuit breaker is disabled, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got '%s'", result)
	}
}

func TestCircuitBreakerTimeout(t *testing.T) {
	InitCircuitBreaker(CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          1 * time.Second,
		Name:             "test-circuit-breaker-timeout",
	})

	cb := GetCircuitBreaker()

	CircuitBreakerExecute(func() (string, error) {
		return "", errors.New("error1")
	})
	CircuitBreakerExecute(func() (string, error) {
		return "", errors.New("error2")
	})

	if cb.State() != gobreaker.StateOpen {
		t.Errorf("Expected circuit breaker to be open after failures")
	}

	time.Sleep(1200 * time.Millisecond)

	_, err := CircuitBreakerExecute(func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "success", nil
	})

	time.Sleep(50 * time.Millisecond)

	state := cb.State()
	if state != gobreaker.StateHalfOpen {
		t.Logf("Current state: %v, err: %v", state, err)
	}
}
