package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pablodz/tracker/tracker"
)

func main() {
	t := tracker.NewTracker(context.TODO(), 100*time.Millisecond)
	defer t.Stop()

	for i := 0; i < 3; i++ {
		go func(id int) {
			t.Start(fmt.Sprintf("worker%d", id))
			defer t.Done(fmt.Sprintf("worker%d", id))

			time.Sleep(time.Duration(id+1) * time.Second)
		}(i)
	}

	go func() {
		t.Start("logger")
		defer t.Done("logger")

		time.Sleep(1 * time.Second)
	}()

	for msg := range t.Reports() {
		println(msg)
	}

	log.Printf("Tracker stopped")
}
