package utils

import (
	"sync"

	"ddbt/fs"
)

const NumberWorkers = 4

func ProcessFiles(files []*fs.File, f func(file *fs.File)) {
	var wait sync.WaitGroup

	c := make(chan *fs.File, len(files))

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
