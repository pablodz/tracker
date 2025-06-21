package tracker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTracker_NoDeadlock_Simple(t *testing.T) {
	tr := NewTracker(context.TODO(), 10*time.Millisecond)
	tr.Start("a")
	tr.Done("a")
	tr.Stop()
	for range tr.Reports() {
		// drain
	}
}

func TestTracker_NoDeadlock_Concurrent(t *testing.T) {
	tr := NewTracker(context.TODO(), 1*time.Millisecond)
	names := []string{"a", "b", "c", "d"}
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				tr.Start(n)
				tr.Done(n)
			}
		}(name)
	}
	wg.Wait()
	tr.Stop()
	for range tr.Reports() {
		// drain
	}
}

func TestTracker_StopIdempotent(t *testing.T) {
	tr := NewTracker(context.TODO(), 5*time.Millisecond)
	tr.Start("x")
	tr.Done("x")
	tr.Stop()
	tr.Stop()
	tr.Stop()
	for range tr.Reports() {
		// drain
	}
}

func TestTracker_ReportsClosed(t *testing.T) {
	tr := NewTracker(context.TODO(), 2*time.Millisecond)
	tr.Start("foo")
	tr.Done("foo")
	tr.Stop()
	select {
	case _, ok := <-tr.Reports():
		if !ok {
			return
		}
	case <-time.After(1 * time.Second):
		t.Fatal("reports channel not closed in time")
	}
}

func TestTracker_RapidStartDone(t *testing.T) {
	tr := NewTracker(context.TODO(), 1*time.Millisecond)
	for i := 0; i < 1000; i++ {
		tr.Start("g")
		tr.Done("g")
	}
	tr.Stop()
	for range tr.Reports() {
		// drain
	}
}

func TestTracker_ManyGoroutinesNotClosed(t *testing.T) {
	tr := NewTracker(context.TODO(), 1*time.Millisecond)
	n := 1000
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tr.Start(fmt.Sprintf("task-%d", i))
			// No Done call
		}(i)
	}
	wg.Wait()
	tr.Stop()
	count := 0
	for msg := range tr.Reports() {
		if msg != "" {
			count++
		}
	}
	if count == 0 {
		t.Fatal("expected some reports for unclosed tasks")
	}
}
