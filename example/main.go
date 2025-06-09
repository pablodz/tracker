package main

import (
	"log"
	"time"

	"github.com/pablodz/tracker/tracker"
)

func main() {
	tracker := tracker.NewTracker(1 * time.Second)
	defer tracker.Stop()

	for i := 0; i < 3; i++ {
		go func(id int) {
			tracker.Start("worker")
			defer tracker.Done("worker")

			time.Sleep(time.Duration(id+1) * time.Second)
		}(i)
	}

	go func() {
		tracker.Start("logger")
		defer tracker.Done("logger")

		for i := 0; i < 3; i++ {
			time.Sleep(2 * time.Second)
		}
	}()

	for msg := range tracker.Reports {
		println(msg)
	}

	log.Printf("Tracker stopped")
}
