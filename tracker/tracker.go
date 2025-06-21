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
}

func NewTracker(ctx context.Context, reportInterval time.Duration) *Tracker {
	t := &Tracker{
		running: make(map[string]struct{}),
		reports: make(chan string, 100),
	}
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(t.reports)
				return
			case <-ticker.C:
				t.reportSummary()
			}
		}
	}()
	return t
}

func (t *Tracker) Start(name string) {
	t.mu.Lock()
	t.running[name] = struct{}{}
	t.mu.Unlock()
	t.report("▶", name)
}

func (t *Tracker) Done(name string) {
	t.mu.Lock()
	delete(t.running, name)
	t.mu.Unlock()
	t.report("■", name)
}

func (t *Tracker) report(kind, name string) {
	t.mu.Lock()
	total := len(t.running)
	t.mu.Unlock()
	msg := fmt.Sprintf("[T=%d] %s %s", total, kind, name)
	select {
	case t.reports <- msg:
	default:
	}
}

func (t *Tracker) reportSummary() {
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

func (t *Tracker) Reports() <-chan string {
	return t.reports
}

func (t *Tracker) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
}
