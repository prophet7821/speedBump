package speedBump

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	requestLimit   int
	windowLength   time.Duration
	keyFn          KeyFunc
	limitCounter   LimitCounter
	onRequestLimit http.HandlerFunc
	mu             sync.Mutex
}

type localCounter struct {
	counter      map[uint64]*count
	windowLength time.Duration
	lastEvict    time.Time
	mu           sync.Mutex
}

type count struct {
	count     int
	updatedAt time.Time
}

func NewRateLimiter(requestLimit int, windowLength time.Duration, options ...Option) *rateLimiter {
	return newRateLimiter(requestLimit, windowLength, options...)
}

func newRateLimiter(requestLimit int, windowLength time.Duration, options ...Option) *rateLimiter {
	limiter := &rateLimiter{
		requestLimit: requestLimit,
		windowLength: windowLength,
	}

	for _, option := range options {
		option(limiter)
	}

	if limiter.keyFn == nil {
		limiter.keyFn = func(r *http.Request) (string, error) {
			return "*", nil
		}
	}

	if limiter.limitCounter == nil {
		limiter.limitCounter = &localCounter{
			counters:     make(map[uint64]*count),
			windowLength: windowLength,
		}
	}

	limiter.limitCounter.Config(requestLimit, windowLength)

	if limiter.onRequestLimit == nil {
		limiter.onRequestLimit = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		}
	}

	return limiter
}

func (l *rateLimiter) Counter() LimitCounter {
	return l.limitCounter
}

func (l *rateLimiter) Status(key string) (bool, float64, error) {
	t := time.Now().UTC()
	currentWindow := t.Truncate(l.windowLength)
	previousWindow := currentWindow.Add(-l.windowLength)

	currCount, prevCount, err := l.limitCounter.Get(key, currentWindow, previousWindow)
	if err != nil {
		return false, 0, err
	}

	diff := t.Sub(currentWindow)
	rate := float64(prevCount)*(float64(l.windowLength)-float64(diff))/float64(l.windowLength) + float64(currCount)

	if rate > float64(l.requestLimit) {
		return false, rate, nil
	}

	return true, rate, nil
}

func (l *rateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := l.keyFn(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		}

		currentWindow := time.Now().UTC().Truncate(l.windowLength)

		w.Header().Set("X-RateLimit", fmt.Sprintf("%d", l.requestLimit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", 0))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", currentWindow.Add(l.windowLength).Unix()))

		l.mu.Lock()

		_, rate, err := l.Status(key)
		if err != nil {
			l.mu.Unlock()
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		}

		nrate := int(math.Round(rate))

		if l.requestLimit > nrate {
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", l.requestLimit-nrate))
		}

		if nrate >= l.requestLimit {
			l.mu.Unlock()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(l.windowLength.Seconds())))
			l.onRequestLimit(w, r)
			return
		}

		err = l.limitCounter.IncBy(key, currentWindow, getInC(r.Context()))
		if err != nil {
			l.mu.Unlock()
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		}

		l.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func (c *localCounter) Config(requestLimit int, windowLength time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.windowLength = windowLength
}

func (c *localCounter) Inc(key string, currentWindow time.Time) error {
	return c.IncBy(key, currentWindow, 1)
}
