package speedBump

import (
	"fmt"
	"github.com/cespare/xxhash/v2"
	"math"
	"net/http"
	"sync"
	"time"
)

type LimitCounter interface {
	Config(windowLength time.Duration)
	Inc(key string, currentWindow time.Time) error
	IncBy(key string, currentWindow time.Time, n int) error
	Get(key string, currentWindow time.Time, previousWindow time.Time) (int, int, error)
}

type rateLimit struct {
	requestLimit   int
	windowLength   time.Duration
	keyFn          KeyFunc
	onRequestLimit http.HandlerFunc
	limitCounter   LimitCounter
	mu             sync.Mutex
}

type localCounter struct {
	counters     map[uint64]*count
	windowLength time.Duration
	lastEvict    time.Time
	mu           sync.Mutex
}

type count struct {
	value     int
	updatedAt time.Time
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

	if limiter.onRequestLimit == nil {
		limiter.onRequestLimit = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Rate limit Exceeded", http.StatusTooManyRequests)
		}
	}

	if limiter.limitCounter == nil {
		limiter.limitCounter = &localCounter{
			counters: make(map[uint64]*count),
		}
	}

	limiter.limitCounter.Config(windowLength)

	return limiter
}

func (limiter *rateLimit) Status(key string) (bool, float64, error) {
	t := time.Now().UTC()

	currentWindow := t.Truncate(limiter.windowLength)
	previousWindow := currentWindow.Add(-limiter.windowLength)

	currCount, prevCount, err := limiter.limitCounter.Get(key, currentWindow, previousWindow)
	if err != nil {
		return false, 0, err
	}

	diff := t.Sub(currentWindow)
	rate := float64(prevCount)*(float64(limiter.windowLength)-float64(diff))/float64(limiter.windowLength) + float64(currCount)

	if rate > float64(limiter.requestLimit) {
		return false, rate, nil
	}

	return true, rate, nil
}

func (limiter *rateLimit) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := limiter.keyFn(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusPreconditionRequired)
		}

		currentWindow := time.Now().UTC().Truncate(limiter.windowLength)

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.requestLimit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", 0))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", currentWindow.Add(limiter.windowLength).Unix()))

		_, rate, err := limiter.Status(key)

		limiter.mu.Lock()
		if err != nil {
			limiter.mu.Unlock()
			http.Error(w, err.Error(), http.StatusPreconditionRequired)
			return
		}

		nrate := int(math.Round(rate))

		if nrate < limiter.requestLimit {
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.requestLimit-nrate))
		}

		if nrate >= limiter.requestLimit {
			limiter.mu.Unlock()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(limiter.windowLength.Seconds())))
			limiter.onRequestLimit(w, r)
			return
		}

		err = limiter.limitCounter.Inc(key, currentWindow)
		if err != nil {
			limiter.mu.Unlock()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		limiter.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
func (l *localCounter) Config(windowLength time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.windowLength = windowLength
}

func (l *localCounter) Inc(key string, currentWindow time.Time) error {
	return l.IncBy(key, currentWindow, 1)
}

func (l *localCounter) IncBy(key string, currentWindow time.Time, n int) error {
	l.evict()

	l.mu.Lock()
	defer l.mu.Unlock()

	hkey := LimitCounterKey(key, currentWindow)

	v, ok := l.counters[hkey]
	if !ok {
		v = &count{}
		l.counters[hkey] = v
	}

	v.value += n
	v.updatedAt = time.Now()

	return nil
}

func (l *localCounter) Get(key string, currentWindow time.Time, previousWindow time.Time) (int, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	curr, ok := l.counters[LimitCounterKey(key, currentWindow)]
	if !ok {
		curr = &count{value: 0, updatedAt: time.Now()}
	}

	prev, ok := l.counters[LimitCounterKey(key, previousWindow)]
	if !ok {
		prev = &count{value: 0, updatedAt: time.Now()}
	}

	return curr.value, prev.value, nil
}

func (l *localCounter) evict() {
	l.mu.Lock()
	defer l.mu.Unlock()

	d := l.windowLength * 3

	if time.Since(l.lastEvict) < d {
		return
	}

	l.lastEvict = time.Now()

	for k, v := range l.counters {
		if time.Since(v.updatedAt) > d {
			delete(l.counters, k)
		}
	}
}

func LimitCounterKey(key string, t time.Time) uint64 {
	hash := xxhash.New()
	hash.WriteString(key)
	hash.WriteString(fmt.Sprintf("%d", t.Unix()))
	return hash.Sum64()
}
