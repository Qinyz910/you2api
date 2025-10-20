package reliability

import (
    "sync"
)

var (
    youCB     *CircuitBreaker
    youCBOnce sync.Once
)

const YouUpstreamLabel = "you_streamingSearch"

func YouUpstreamCB() *CircuitBreaker {
    youCBOnce.Do(func() {
        window := getEnvDuration("CB_WINDOW_MS", 30000)
        minSamples := getEnvInt("CB_MIN_SAMPLES", 10)
        thresholdPct := getEnvInt("CB_FAILURE_THRESHOLD_PERCENT", 50)
        cooldown := getEnvDuration("CB_OPEN_COOLDOWN_MS", 10000)
        threshold := float64(thresholdPct) / 100.0
        youCB = NewCircuitBreaker(window, minSamples, threshold, cooldown, YouUpstreamLabel)
    })
    return youCB
}

// Optionally expose a helper to record outcome with reason
func RecordUpstreamResult(success bool, reason string) {
    cb := YouUpstreamCB()
    if success {
        cb.OnSuccess()
    } else {
        cb.OnFailure(reason)
    }
}
