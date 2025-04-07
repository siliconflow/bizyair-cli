package lib

import (
	"time"
)

func Throttle(fn func(x int), wait time.Duration) func(x int) {
	lastTime := time.Now()
	return func(x int) {
		now := time.Now()
		if now.Sub(lastTime) >= wait {
			fn(x)
			lastTime = now
		}
	}
}
