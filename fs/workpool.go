package fs

import (
	"fmt"
	"sync"

	"ddbt/config"
	"ddbt/utils"
)

// Process the given file list through `f`. If a progress bar is given, then it will show the stauts line as we go
func ProcessFiles(files []*File, f func(file *File), pb *utils.ProgressBar) {
	var wait sync.WaitGroup

	c := make(chan *File, len(files))

	worker := func() {
		var statusRow *utils.StatusRow

		if pb != nil {
			statusRow = pb.NewStatusRow()
		}

		for file := range c {
			if statusRow != nil {
				statusRow.Update(fmt.Sprintf("Running %s", file.Name))
			}

			f(file)
			wait.Done()

			if statusRow != nil {
				statusRow.SetIdle()
			}
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
