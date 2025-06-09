package tracker

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type counter struct {
	value int32
}

func (c *counter) inc() {
	atomic.AddInt32(&c.value, 1)
}

func (c *counter) dec() {
	atomic.AddInt32(&c.value, -1)
}

func (c *counter) get() int32 {
	return atomic.LoadInt32(&c.value)
}

type Tracker struct {
	counters sync.Map
	interval time.Duration
	stopCh   chan struct{}
}

// NewTracker creates a new Tracker with a monitoring interval.
func NewTracker(reportInterval time.Duration) *Tracker {
	t := &Tracker{
		interval: reportInterval,
		stopCh:   make(chan struct{}),
	}
	go t.monitor()
	return t
}

func (t *Tracker) Start(name string) {
	val, _ := t.counters.LoadOrStore(name, &counter{})
	val.(*counter).inc()
	t.report()
}

func (t *Tracker) Done(name string) {
	val, ok := t.counters.Load(name)
	if ok {
		val.(*counter).dec()
		t.report()
	}
}

func (t *Tracker) Snapshot() map[string]int32 {
	snapshot := make(map[string]int32)
	t.counters.Range(func(key, value any) bool {
		name := key.(string)
		cnt := value.(*counter).get()
		if cnt > 0 {
			snapshot[name] = cnt
		}
		return true
	})
	return snapshot
}

func (t *Tracker) monitor() {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.report()
		case <-t.stopCh:
			return
		}
	}
}

func (t *Tracker) report() {
	snap := t.Snapshot()
	fmt.Printf("[tracker] Total: %d | ", runtime.NumGoroutine())
	if len(snap) == 0 {
		fmt.Printf("NO_GOROUTINES")
	} else {
		for name, count := range snap {
			fmt.Printf("%s=%d ", name, count)
		}
	}
	fmt.Println()
}

func (t *Tracker) Stop() {
	close(t.stopCh)
	t.report()
}
