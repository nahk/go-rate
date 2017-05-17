package rate

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

// A RateLimiter limits the rate at which an action can be performed.  It
// applies neither smoothing (like one could achieve in a token bucket system)
// nor does it offer any conception of warmup, wherein the rate of actions
// granted are steadily increased until a steady throughput equilibrium is
// reached.
type RateLimiter struct {
	limit     int
	remaining int
	resetAt   time.Time
}

// New creates a new rate limiter for the limit and resetAt.
func New(limit int, resetAt time.Time) *RateLimiter {
	lim := &RateLimiter{
		limit:     limit,
		remaining: limit,
		resetAt:   resetAt,
	}
	return lim
}

// Update resetAt with an extra second as safety
func (r *RateLimiter) Update(header http.Header) error {
	reset, err := strconv.ParseInt(header.Get("X-Ratelimit-Reset"), 10, 64)
	if err != nil {
		return errors.New("X-Ratelimit-Reset header is not defined")
	}
	r.resetAt = time.Unix(reset+1, 0)

	remaining, err := strconv.Atoi(header.Get("X-Ratelimit-Remaining"))
	if err != nil {
		return errors.New("X-Ratelimit-Remaining header is not defined")
	}
	r.remaining = remaining

	limit, err := strconv.Atoi(header.Get("X-Ratelimit-Limit"))
	if err != nil {
		return errors.New("X-Ratelimit-Limit header is not defined")
	}
	r.limit = limit
	return nil
}

// Wait blocks if the rate limit has been reached.  Wait offers no guarantees
// of fairness for multiple actors if the allowed rate has been temporarily
// exhausted.
func (r *RateLimiter) Wait() (ok bool, pause time.Duration) {
	ok, pause = r.Try()
	if !ok {
		time.Sleep(pause)
	}
	return
}

// Try returns true if under the rate limit, or false if over and the
// remaining time before the rate limit expires.
func (r *RateLimiter) Try() (bool, time.Duration) {
	if r.remaining > 0 {
		return true, 0
	}
	return false, r.resetAt.Sub(time.Now())
}

func (r *RateLimiter) Remaining() int {
	return r.remaining
}
