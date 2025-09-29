package lib

import (
	"time"
)

func Throttle(fn func(x int64), wait time.Duration) func(x int64) {
	lastTime := time.Now()
	return func(x int64) {
		now := time.Now()
		if now.Sub(lastTime) >= wait {
			fn(x)
			lastTime = now
		}
	}
}
