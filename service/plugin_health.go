package service

import (
	"sort"
	"sync"
	"time"
)

const (
	defaultCircuitFailureThreshold = 3
	defaultCircuitCooldown         = 2 * time.Minute
)

// PluginHealth describes the runtime health and circuit-breaker state of one plugin.
type PluginHealth struct {
	Name                string    `json:"name"`
	Requests            int64     `json:"requests"`
	Successes           int64     `json:"successes"`
	Failures            int64     `json:"failures"`
	Timeouts            int64     `json:"timeouts"`
	EmptyResults        int64     `json:"empty_results"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	AverageLatencyMS    float64   `json:"average_latency_ms"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	LastFailureAt       time.Time `json:"last_failure_at,omitempty"`
	CircuitOpen         bool      `json:"circuit_open"`
	CircuitOpenedAt     time.Time `json:"circuit_opened_at,omitempty"`
	CircuitRetryAt      time.Time `json:"circuit_retry_at,omitempty"`
}

type pluginHealthState struct {
	PluginHealth
	totalLatency time.Duration
	halfOpen     bool
}

// PluginHealthTracker stores process-local plugin metrics and circuit-breaker state.
type PluginHealthTracker struct {
	mu               sync.RWMutex
	states           map[string]*pluginHealthState
	failureThreshold int
	cooldown         time.Duration
	now              func() time.Time
}

func NewPluginHealthTracker(failureThreshold int, cooldown time.Duration) *PluginHealthTracker {
	if failureThreshold <= 0 {
		failureThreshold = defaultCircuitFailureThreshold
	}
	if cooldown <= 0 {
		cooldown = defaultCircuitCooldown
	}
	return &PluginHealthTracker{
		states:           make(map[string]*pluginHealthState),
		failureThreshold: failureThreshold,
		cooldown:         cooldown,
		now:              time.Now,
	}
}

func (t *PluginHealthTracker) stateLocked(name string) *pluginHealthState {
	state := t.states[name]
	if state == nil {
		state = &pluginHealthState{PluginHealth: PluginHealth{Name: name}}
		t.states[name] = state
	}
	return state
}

// Allow reports whether a plugin may run. After cooldown, one half-open probe is allowed.
func (t *PluginHealthTracker) Allow(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.stateLocked(name)
	if !state.CircuitOpen {
		return true
	}
	now := t.now()
	if now.Before(state.CircuitRetryAt) || state.halfOpen {
		return false
	}
	state.halfOpen = true
	return true
}

func (t *PluginHealthTracker) RecordSuccess(name string, latency time.Duration, resultCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.stateLocked(name)
	state.Requests++
	state.Successes++
	if resultCount == 0 {
		state.EmptyResults++
	}
	state.totalLatency += latency
	state.AverageLatencyMS = float64(state.totalLatency.Microseconds()) / 1000 / float64(state.Requests)
	state.LastSuccessAt = t.now()
	state.ConsecutiveFailures = 0
	state.CircuitOpen = false
	state.CircuitOpenedAt = time.Time{}
	state.CircuitRetryAt = time.Time{}
	state.halfOpen = false
}

func (t *PluginHealthTracker) RecordFailure(name string, latency time.Duration, timedOut bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.stateLocked(name)
	state.Requests++
	state.Failures++
	if timedOut {
		state.Timeouts++
	}
	state.totalLatency += latency
	state.AverageLatencyMS = float64(state.totalLatency.Microseconds()) / 1000 / float64(state.Requests)
	now := t.now()
	state.LastFailureAt = now
	state.ConsecutiveFailures++
	state.halfOpen = false
	if state.ConsecutiveFailures >= t.failureThreshold {
		state.CircuitOpen = true
		state.CircuitOpenedAt = now
		state.CircuitRetryAt = now.Add(t.cooldown)
	}
}

func (t *PluginHealthTracker) Snapshot() []PluginHealth {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]PluginHealth, 0, len(t.states))
	for _, state := range t.states {
		result = append(result, state.PluginHealth)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

var globalPluginHealthTracker = NewPluginHealthTracker(defaultCircuitFailureThreshold, defaultCircuitCooldown)

func GetPluginHealthSnapshot() []PluginHealth {
	return globalPluginHealthTracker.Snapshot()
}
