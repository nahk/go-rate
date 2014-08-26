package rate

import (
	"container/list"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// A RateLimiter limits the rate at which an action can be performed.  It
// applies neither smoothing (like one could achieve in a token bucket system)
// nor does it offer any conception of warmup, wherein the rate of actions
// granted are steadily increased until a steady throughput equilibrium is
// reached.
type RateLimiter struct {
	Limit   int
	ResetAt time.Time
	mtx     sync.Mutex
	times   list.List
}

// New creates a new rate limiter for the limit and resetAt.
func New(limit int, resetAt time.Time) *RateLimiter {
	lim := &RateLimiter{
		Limit:   limit,
		ResetAt: resetAt,
	}
	lim.times.Init()
	return lim
}

// Update resetAt with an extra second as safety
func (r *RateLimiter) Update(header http.Header) error {
	reset, err := strconv.ParseInt(header.Get("X-Ratelimit-Reset"), 10, 64)
	if err != nil {
		return errors.New("X-Ratelimit-Reset header is not defined")
	}
	r.ResetAt = time.Unix(reset+1, 0)

	remaining, err := strconv.Atoi(header.Get("X-Ratelimit-Remaining"))
	if err != nil {
		return errors.New("X-Ratelimit-Remaining header is not defined")
	}
	r.Limit = remaining
	return nil
}

// Wait blocks if the rate limit has been reached.  Wait offers no guarantees
// of fairness for multiple actors if the allowed rate has been temporarily
// exhausted.
func (r *RateLimiter) Wait() {
	for {
		ok, remaining := r.Try()
		if ok {
			break
		}
		time.Sleep(remaining)
	}
}

// Try returns true if under the rate limit, or false if over and the
// remaining time before the rate limit expires.
func (r *RateLimiter) Try() (bool, time.Duration) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	now := time.Now()
	if l := r.times.Len(); l < r.Limit {
		r.times.PushBack(now)
		return true, 0
	}
	frnt := r.times.Front()
	if diff := r.ResetAt.Sub(now); diff.Seconds() > 0 {
		return false, diff
	}
	frnt.Value = now
	r.times.MoveToBack(frnt)
	return true, 0
}
