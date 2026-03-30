package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

var (
	cb   *gobreaker.CircuitBreaker
	cbMu sync.RWMutex
)

type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	Name             string
}

func InitCircuitBreaker(cfg CircuitBreakerConfig) {
	cbMu.Lock()
	defer cbMu.Unlock()

	cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: uint32(cfg.SuccessThreshold),
		Interval:    0,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(cfg.FailureThreshold)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger := zap.L()
			switch to {
			case gobreaker.StateOpen:
				logger.Warn("circuit breaker opened",
					zap.String("name", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			case gobreaker.StateHalfOpen:
				logger.Info("circuit breaker half-open",
					zap.String("name", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			case gobreaker.StateClosed:
				logger.Info("circuit breaker closed",
					zap.String("name", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			}
			RecordCircuitBreakerState(name, float64(to))
		},
	})
}

func GetCircuitBreaker() *gobreaker.CircuitBreaker {
	cbMu.RLock()
	defer cbMu.RUnlock()
	return cb
}

func CircuitBreakerExecute[T any](fn func() (T, error)) (T, error) {
	cbMu.RLock()
	circuitBreaker := cb
	cbMu.RUnlock()

	if circuitBreaker == nil {
		return fn()
	}

	result, err := circuitBreaker.Execute(func() (interface{}, error) {
		return fn()
	})

	if err != nil {
		var zero T
		return zero, err
	}

	return result.(T), nil
}

type HTTPResult struct {
	Response *http.Response
	Body     []byte
	Error    error
}

func CircuitBreakerHTTPExecute(fn func() (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	cbMu.RLock()
	circuitBreaker := cb
	cbMu.RUnlock()

	if circuitBreaker == nil {
		return fn()
	}

	result, err := circuitBreaker.Execute(func() (interface{}, error) {
		resp, body, err := fn()
		return HTTPResult{Response: resp, Body: body, Error: err}, err
	})

	if err != nil {
		return nil, nil, err
	}

	httpResult := result.(HTTPResult)
	return httpResult.Response, httpResult.Body, httpResult.Error
}

type CircuitBreakerMetrics struct {
	State               string
	Failures            uint32
	Successes           uint32
	ConsecutiveFailures uint32
}

func GetCircuitBreakerState() gobreaker.State {
	return gobreaker.StateClosed
}
