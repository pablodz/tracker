//go:build !race

package tracker

import (
	"testing"
	"time"
)

func TestNewTracker_BasicUsage(t *testing.T) {
	tr := NewTracker(50 * time.Millisecond)
	defer tr.Stop()

	tr.Start("foo")
	tr.Start("bar")
	tr.Start("foo")

	snap := tr.Snapshot()
	if snap["foo"] != 2 {
		t.Errorf("expected foo count 2, got %d", snap["foo"])
	}
	if snap["bar"] != 1 {
		t.Errorf("expected bar count 1, got %d", snap["bar"])
	}

	tr.Done("foo")
	snap = tr.Snapshot()
	if snap["foo"] != 1 {
		t.Errorf("expected foo count 1 after Done, got %d", snap["foo"])
	}

	tr.Done("foo")
	tr.Done("bar")
	snap = tr.Snapshot()
	if _, ok := snap["foo"]; ok {
		t.Errorf("expected foo to be absent after all Done, got %v", snap["foo"])
	}
	if _, ok := snap["bar"]; ok {
		t.Errorf("expected bar to be absent after all Done, got %v", snap["bar"])
	}

}

func TestNewTracker_Stop(t *testing.T) {
	tr := NewTracker(10 * time.Millisecond)
	tr.Start("baz")
	tr.Stop()
	// Should not panic or deadlock
}

func TestTracker_ReportsChannel(t *testing.T) {
	tr := NewTracker(10 * time.Millisecond)
	defer tr.Stop()

	tr.Start("foo")
	tr.Start("bar")
	tr.Done("foo")
	tr.Done("bar")

	timeout := time.After(200 * time.Millisecond)
	got := []string{}
loop:
	for {
		select {
		case msg := <-tr.Reports:
			got = append(got, msg)
		case <-timeout:
			break loop
		}
	}

	if len(got) == 0 {
		t.Error("expected some reports, got none")
	}
	// Optionally, print for manual inspection
	for _, msg := range got {
		t.Log(msg)
	}
}
