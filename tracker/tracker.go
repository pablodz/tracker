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
	stopOnce sync.Once // ensure Stop is called only once
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
		// Only report done if count is non-negative
		if val.(*counter).get() >= 0 {
			t.report(reportDone, name)
		}
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
	emptyReported := false

	for {
		select {
		case <-ticker.C:
			t.report(reportSummary, "")
			snap := t.Snapshot()
			if len(snap) == 0 {
				if !emptyReported {
					emptyReported = true
					t.Stop()
				}
				return
			}
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

	// Removed timestamp from report output
	var msg string
	switch rtype {
	case reportStart:
		msg = fmt.Sprintf("[T=%d] ▶ %s", total, taskName)
	case reportDone:
		msg = fmt.Sprintf("[T=%d] ■ %s", total, taskName)
	case reportSummary:
		if len(snap) == 0 {
			msg = "[T=0] ≡ (empty)"
		} else {
			msg = fmt.Sprintf("[T=%d] ≡", total)
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
	t.stopOnce.Do(func() {
		close(t.stopCh)
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
			}
		}
		// Only report summary if not already empty
		if len(snap) > 0 {
			t.report(reportSummary, "")
		}
	})
}
