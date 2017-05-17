package rate

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestRateLimiter_Update(t *testing.T) {
	start := time.Now()
	plus1Sec := start.Add(time.Second * time.Duration(1))
	plus3Sec := start.Add(time.Second * time.Duration(3))

	limit := 5
	resetAt := plus3Sec
	limiter := New(limit, resetAt)

	header := http.Header{}
	header.Add("X-Ratelimit-Reset", strconv.FormatInt(plus1Sec.Unix(), 10))
	header.Add("X-Ratelimit-Limit", strconv.Itoa(10))
	header.Add("X-Ratelimit-Remaining", strconv.Itoa(10))
	if err := limiter.Update(header); err != nil {
		t.Error(err)
	}

	if limiter.resetAt.Unix() != plus1Sec.Unix()+1 {
		t.Error("The limiter hasn't been updated with the proper reset time")
	}
	if limiter.limit != 10 {
		t.Error("The limiter hasn't been updated with the proper limit")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	start := time.Now()
	loops := 2
	wait := 3

	limit := 3
	limiter := New(limit, start.Add(time.Second*time.Duration(5)))

	resetAt := start
	for k := 0; k < loops; k++ {
		resetAt = resetAt.Add(time.Second * time.Duration(wait))
		for i := 1; i <= limit; i++ {
			limiter.Wait()

			header := http.Header{}
			header.Add("X-Ratelimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
			header.Add("X-Ratelimit-Limit", strconv.Itoa(limit))
			header.Add("X-Ratelimit-Remaining", strconv.Itoa(limit-i))
			limiter.Update(header)
		}
	}

	if time.Now().Sub(start) < time.Second*time.Duration(wait*(loops-1)) {
		t.Error("The limiter didn't block when it should have")
	}
}
