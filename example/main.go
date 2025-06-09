package main

import (
	"time"

	"github.com/pablodz/tracker/tracker"
)

func main() {
	tracker := tracker.NewTracker(1 * time.Second)
	defer tracker.Stop()

	for i := 0; i < 5; i++ {
		go func(id int) {
			tracker.Start("worker")
			defer tracker.Done("worker")

			time.Sleep(time.Duration(id+1) * time.Second)
		}(i)
	}

	// Additional goroutine type: "logger"
	go func() {
		tracker.Start("logger")
		defer tracker.Done("logger")

		for i := 0; i < 3; i++ {
			time.Sleep(2 * time.Second)
		}
	}()

	time.Sleep(10 * time.Second)
}
