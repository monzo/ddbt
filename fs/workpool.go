package fs

import (
	"sync"
)

const NumberWorkers = 8

func ProcessFiles(files []*File, f func(file *File)) {
	var wait sync.WaitGroup

	c := make(chan *File, len(files))

	worker := func() {
		for file := range c {
			f(file)
			wait.Done()
		}
	}

	for i := 0; i < NumberWorkers; i++ {
		go worker()
	}

	wait.Add(len(files))
	for _, file := range files {
		c <- file
	}

	wait.Wait()
}
