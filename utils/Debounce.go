package utils

import (
	"sync"
	"time"
)

type DebouncedFunction func()

type debounce struct {
	m     sync.Mutex
	timer *time.Timer
}

func Debounce(f DebouncedFunction, duration time.Duration) DebouncedFunction {
	debouncer := &debounce{}

	execute := func() {
		debouncer.m.Lock()
		defer debouncer.m.Unlock()

		f()
		debouncer.timer = nil
	}

	return func() {
		debouncer.m.Lock()
		defer debouncer.m.Unlock()

		if debouncer.timer != nil {
			debouncer.timer.Stop()
		}

		debouncer.timer = time.AfterFunc(duration, execute)
	}
}
