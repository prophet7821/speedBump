package speedBump

import (
	"net/http"
	"strings"
	"time"
)

type KeyFunc func(r *http.Request) (string, error)

func Limit(requestLimit int, windowLength time.Duration, options ...Option) func(next http.Handler) http.Handler {
	return NewRateLimiter(requestLimit, windowLength, options...).Handler

}

func LimitAll(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return Limit(requestLimit, windowLength)
}

func LimitByIp(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return Limit(requestLimit, windowLength, WithKeyFuncs(KeyByIP))
}

func KeyByIP(r *http.Request) (string, error) {
	ip, _, err := GetIP(r)
	if err != nil {
		ip = r.RemoteAddr
	}

	return ip, nil
}

func KeyByEndpoint(r *http.Request) (string, error) {
	return r.URL.Path, nil
}

func WithKeyFuncs(keyFuncs ...KeyFunc) KeyFunc {
	return func(r *http.Request) (string, error) {
		var key strings.Builder
		for _, keyFunc := range keyFuncs {
			k, err := keyFunc(r)
			if err != nil {
				return "", err
			}
			key.WriteString(k)
			key.WriteRune(':')
		}

		return key.String(), nil
	}
}
