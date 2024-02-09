package speedBump

import (
	"net/http"
	"time"
)

type LimitCounter interface {
	Config(requestLimit int, windowLength time.Duration)
	Inc(key string, currentWindow time.Time) (int, error)
	IncBy(key string, currentWindow time.Time, n int) (int, error)
	Get(key string, currentWindow time.Time, previousWindow time.Time) (int, int, error)
}

type rateLimit struct {
	requestLimit   int
	windowLength   time.Duration
	keyFn          KeyFunc
	onRequestLimit http.HandlerFunc
	limitCounter   LimitCounter
}

type localCounter struct {
	counters     map[string]*interface{}
	windowLength time.Duration
}

func NewRateLimiter(requestLimit int, windowLength time.Duration, options ...Option) *rateLimit {
	return newRateLimiter(requestLimit, windowLength, options...)
}

func newRateLimiter(requestLimit int, windowLength time.Duration, options ...Option) *rateLimit {
	limiter := &rateLimit{
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
			counters:     make(map[string]*counter),
			windowLength: windowLength,
		}
	}
}
