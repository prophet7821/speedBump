package speedBump

import (
	"net"
	"net/http"
	"strings"
	"time"
)

type Option func(limiter *rateLimit)

type KeyFunc func(r *http.Request) (string, error)

func Limit(requestLimit int, windowLength time.Duration, options ...Option) func(next http.Handler) http.Handler {
	return NewRateLimiter(requestLimit, windowLength, options...).Handler
}

func LimitAll(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return Limit(requestLimit, windowLength)
}

func LimitByIP(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return Limit(requestLimit, windowLength, WithKeyFuncs(KeyByIP))
}

func KeyByIP(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	return ip, nil
}

func KeyByEndpoint(r *http.Request) (string, error) {
	return r.URL.Path, nil
}

func WithKeyFuncs(keyFuncs ...KeyFunc) Option {
	return func(limiter *rateLimit) {
		if len(keyFuncs) > 0 {
			limiter.keyFn = composeKeyFn(keyFuncs...)
		}
	}
}

func composeKeyFn(keyFuncs ...KeyFunc) KeyFunc {
	return func(r *http.Request) (string, error) {
		var key strings.Builder
		for _, fn := range keyFuncs {
			k, err := fn(r)
			if err != nil {
				return "", err
			}

			key.WriteString(k)
			key.WriteRune(':')
		}

		return key.String(), nil
	}
}

func WithLimitHandler(h http.HandlerFunc) Option {
	return func(limiter *rateLimit) {
		limiter.onRequestLimit = h
	}
}
