package speedBump

import (
	"net/http"
	"time"
)

func Limit(requestLimit int, windowLength time.Duration, options ...Option) func(next http.Handler) http.Handler {
	return NewRateLimiter(requestLimit, windowLength, options...).Handler

}
