package tracker

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Tracker struct {
	mu      sync.Mutex
	running map[string]struct{}
	reports chan string
	cancel  context.CancelFunc
	done    chan struct{} // signal for shutdown
	once    sync.Once     // ensure Stop is only called once
}

func NewTracker(ctx context.Context, reportInterval time.Duration) *Tracker {
	t := &Tracker{
		running: make(map[string]struct{}),
		reports: make(chan string),
		done:    make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()
		defer close(t.reports) // close reports channel when goroutine exits
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.done:
				return
			case <-ticker.C:
				t.reportSummary()
			}
		}
	}()
	return t
}

func (t *Tracker) Start(name string) {
	if t == nil || t.isDone() {
		return
	}
	t.mu.Lock()
	t.running[name] = struct{}{}
	t.mu.Unlock()
	t.report("▶", name)
}

func (t *Tracker) Done(name string) {
	if t == nil || t.isDone() {
		return
	}
	t.mu.Lock()
	delete(t.running, name)
	t.mu.Unlock()
	t.report("■", name)
}

func (t *Tracker) report(kind, name string) {
	if t == nil || t.isDone() {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			// Channel closed, ignore send
		}
	}()
	t.mu.Lock()
	total := len(t.running)
	t.mu.Unlock()
	msg := fmt.Sprintf("[T=%d] %s %s", total, kind, name)
	select {
	case <-t.done:
		// Tracker is done, don't send
		return
	case t.reports <- msg:
	default:
	}
}

func (t *Tracker) reportSummary() {
	if t == nil || t.isDone() {
		return
	}
	t.mu.Lock()
	total := len(t.running)
	names := make([]string, 0, total)
	for name := range t.running {
		names = append(names, name)
	}
	t.mu.Unlock()
	sort.Strings(names)
	if total == 0 {
		msg := "[T=0] No more goroutines. Stopping tracker."
		select {
		case t.reports <- msg:
		default:
		}
		t.Stop()
		return
	}
	msg := fmt.Sprintf("[T=%d] ≡ %v", total, names)
	select {
	case t.reports <- msg:
	default:
	}
}

func (t *Tracker) isDone() bool {
	if t == nil {
		return true
	}
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}

func (t *Tracker) Reports() <-chan string {
	if t == nil {
		return nil
	}
	return t.reports
}

func (t *Tracker) Stop() {
	if t == nil {
		return
	}
	t.once.Do(func() {
		t.mu.Lock()
		select {
		case <-t.done:
			t.mu.Unlock()
			return // already stopped
		default:
			close(t.done)
		}
		t.mu.Unlock()
		if t.cancel != nil {
			t.cancel()
		}
	})
}
