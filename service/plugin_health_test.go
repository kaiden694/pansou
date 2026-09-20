package service

import (
	"sync"
	"testing"
	"time"
)

func TestPluginHealthTrackerOpensCircuitAndRecovers(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	tracker := NewPluginHealthTracker(2, time.Minute)
	tracker.now = func() time.Time { return now }

	tracker.RecordFailure("slow", 100*time.Millisecond, false)
	if !tracker.Allow("slow") {
		t.Fatal("plugin should remain available before threshold")
	}

	tracker.RecordFailure("slow", 200*time.Millisecond, true)
	if tracker.Allow("slow") {
		t.Fatal("plugin should be blocked while circuit is open")
	}

	snapshot := tracker.Snapshot()[0]
	if !snapshot.CircuitOpen || snapshot.Failures != 2 || snapshot.Timeouts != 1 {
		t.Fatalf("unexpected open-circuit snapshot: %#v", snapshot)
	}

	now = now.Add(time.Minute)
	if !tracker.Allow("slow") {
		t.Fatal("one half-open probe should be allowed after cooldown")
	}
	if tracker.Allow("slow") {
		t.Fatal("a second concurrent half-open probe must be rejected")
	}

	tracker.RecordSuccess("slow", 50*time.Millisecond, 1)
	if !tracker.Allow("slow") {
		t.Fatal("successful probe should close the circuit")
	}
	snapshot = tracker.Snapshot()[0]
	if snapshot.CircuitOpen || snapshot.ConsecutiveFailures != 0 || snapshot.Successes != 1 {
		t.Fatalf("unexpected recovered snapshot: %#v", snapshot)
	}
}

func TestPluginHealthTrackerCountsEmptyResultsAndLatency(t *testing.T) {
	tracker := NewPluginHealthTracker(3, time.Minute)
	tracker.RecordSuccess("empty", 100*time.Millisecond, 0)
	tracker.RecordSuccess("empty", 300*time.Millisecond, 2)

	snapshot := tracker.Snapshot()[0]
	if snapshot.Requests != 2 || snapshot.Successes != 2 || snapshot.EmptyResults != 1 {
		t.Fatalf("unexpected counters: %#v", snapshot)
	}
	if snapshot.AverageLatencyMS != 200 {
		t.Fatalf("expected average latency 200ms, got %v", snapshot.AverageLatencyMS)
	}
}

func TestPluginHealthTrackerConcurrentUse(t *testing.T) {
	tracker := NewPluginHealthTracker(1000, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tracker.Allow("parallel") {
				tracker.RecordSuccess("parallel", time.Millisecond, 1)
			}
		}()
	}
	wg.Wait()

	snapshot := tracker.Snapshot()[0]
	if snapshot.Requests != 100 || snapshot.Successes != 100 {
		t.Fatalf("lost concurrent updates: %#v", snapshot)
	}
}
