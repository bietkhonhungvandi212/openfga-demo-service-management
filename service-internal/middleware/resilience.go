package middleware

import (
	"sync"
	"time"

	"github.com/google/uuid"
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

type IdempotencyStore struct {
	mu    sync.RWMutex
	store map[string]idempotencyEntry
	ttl   time.Duration
}

type idempotencyEntry struct {
	statusCode int
	body       []byte
	headers    map[string]string
	expires    time.Time
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	return &IdempotencyStore{
		store: make(map[string]idempotencyEntry),
		ttl:   ttl,
	}
}

func (s *IdempotencyStore) Get(key string) (int, []byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.store[key]
	if !exists || time.Now().After(entry.expires) {
		return 0, nil, false
	}

	return entry.statusCode, entry.body, true
}

func (s *IdempotencyStore) Set(key string, statusCode int, body []byte, headers map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[key] = idempotencyEntry{
		statusCode: statusCode,
		body:       body,
		headers:    headers,
		expires:    time.Now().Add(s.ttl),
	}
}

func (s *IdempotencyStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, entry := range s.store {
		if now.After(entry.expires) {
			delete(s.store, key)
		}
	}
}

type RequestIDGenerator struct{}

func (g RequestIDGenerator) Generate() string {
	return uuid.New().String()
}

var defaultRequestIDGenerator = RequestIDGenerator{}
