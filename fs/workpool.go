package fs

import (
	"sync"

	"ddbt/config"
)

func ProcessFiles(files []*File, f func(file *File)) {
	var wait sync.WaitGroup

	c := make(chan *File, len(files))

	worker := func() {
		for file := range c {
			f(file)
			wait.Done()
		}
	}

	for i := 0; i < config.NumberThreads(); i++ {
		go worker()
	}

	wait.Add(len(files))
	for _, file := range files {
		c <- file
	}

	wait.Wait()
}
