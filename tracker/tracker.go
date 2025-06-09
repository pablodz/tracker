package tracker

import (
	"fmt"
	"sort"
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
	Reports  chan string
}

type reportType int

const (
	reportSummary reportType = iota
	reportStart
	reportDone
)

// NewTracker creates a new Tracker with a monitoring interval.
func NewTracker(reportInterval time.Duration) *Tracker {
	t := &Tracker{
		interval: reportInterval,
		stopCh:   make(chan struct{}),
		Reports:  make(chan string, 100),
	}
	go t.monitor()
	return t
}

func (t *Tracker) Start(name string) {
	val, _ := t.counters.LoadOrStore(name, &counter{})
	val.(*counter).inc()
	t.report(reportStart, name)
}

func (t *Tracker) Done(name string) {
	val, ok := t.counters.Load(name)
	if ok {
		val.(*counter).dec()
		t.report(reportDone, name)
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
			t.report(reportSummary, "")
		case <-t.stopCh:
			return
		}
	}
}

func (t *Tracker) report(rtype reportType, taskName string) {
	snap := t.Snapshot()
	total := int32(0)
	for _, count := range snap {
		total += count
	}

	var msg string
	switch rtype {
	case reportStart:
		msg = fmt.Sprintf("[T=%d] ▶ %s", total, taskName) // ▶ for start
	case reportDone:
		msg = fmt.Sprintf("[T=%d] ■ %s", total, taskName) // ■ for end
	case reportSummary:
		if len(snap) == 0 {
			msg = "[T=0] ≡ (empty)"
		} else {
			msg = fmt.Sprintf("[T=%d] ≡", total) // ≡ for monitor
			names := make([]string, 0, len(snap))
			for name := range snap {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				msg += fmt.Sprintf(" %s:%d", name, snap[name])
			}
		}
	}
	select {
	case t.Reports <- msg:
	default:
		// drop if buffer full
	}
}

func (t *Tracker) Stop() {
	close(t.stopCh)
	// Enhanced: show remaining tracked tasks on stop, if any
	snap := t.Snapshot()
	if len(snap) > 0 {
		names := make([]string, 0, len(snap))
		for name := range snap {
			names = append(names, name)
		}
		sort.Strings(names)
		msg := "[STOP] Remaining tasks:"
		for _, name := range names {
			msg += fmt.Sprintf(" %s:%d", name, snap[name])
		}
		select {
		case t.Reports <- msg:
		default:
			// drop if buffer full
		}
	}
	t.report(reportSummary, "")
}
