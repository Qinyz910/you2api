package reliability

import (
	"sync"
	"time"

	metrics "you2api/metrics"
)

type CircuitBreakerState int

const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

type outcome struct {
	time    time.Time
	success bool
}

type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitBreakerState
	openedAt         time.Time
	lastOpenReason   string
	window           time.Duration
	minSamples       int
	failureThreshold float64
	openCooldown     time.Duration
	// 结果窗口
	results []outcome
	// 标签
	upstreamLabel string
	// 半开仅允许一个探测请求
	halfOpenProbeInFlight bool
}

func NewCircuitBreaker(window time.Duration, minSamples int, failureThreshold float64, openCooldown time.Duration, upstreamLabel string) *CircuitBreaker {
	cb := &CircuitBreaker{
		state:            Closed,
		window:           window,
		minSamples:       minSamples,
		failureThreshold: failureThreshold,
		openCooldown:     openCooldown,
		results:          make([]outcome, 0, minSamples*2),
		upstreamLabel:    upstreamLabel,
	}
	metrics.CircuitBreakerState.WithLabelValues(upstreamLabel).Set(0)
	return cb
}

func (c *CircuitBreaker) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	switch c.state {
	case Open:
		if now.Sub(c.openedAt) >= c.openCooldown {
			c.state = HalfOpen
			c.halfOpenProbeInFlight = false
			metrics.CircuitBreakerState.WithLabelValues(c.upstreamLabel).Set(2)
			// 进入半开
		} else {
			return false
		}
	}
	if c.state == HalfOpen {
		if c.halfOpenProbeInFlight {
			return false
		}
		c.halfOpenProbeInFlight = true
		return true
	}
	return true
}

func (c *CircuitBreaker) OnSuccess() {
	c.record(true, "")
}

func (c *CircuitBreaker) OnFailure(reason string) {
	c.record(false, reason)
}

func (c *CircuitBreaker) record(success bool, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	// prune 过期结果
	cutoff := now.Add(-c.window)
	j := 0
	for _, r := range c.results {
		if r.time.After(cutoff) {
			c.results[j] = r
			j++
		}
	}
	c.results = c.results[:j]
	// 添加新结果
	c.results = append(c.results, outcome{time: now, success: success})

	if c.state == HalfOpen {
		c.halfOpenProbeInFlight = false
		if success {
			// 关闭
			c.state = Closed
			metrics.CircuitBreakerState.WithLabelValues(c.upstreamLabel).Set(0)
		} else {
			// 失败则重新开启
			c.toOpen(reason)
		}
		return
	}

	// 仅在关闭状态评估失败率
	if c.state == Closed {
		total := 0
		fails := 0
		for _, r := range c.results {
			total++
			if !r.success {
				fails++
			}
		}
		if total >= c.minSamples {
			failureRate := float64(fails) / float64(total)
			if failureRate >= c.failureThreshold && fails > 0 {
				c.toOpen(reason)
			}
		}
	}
}

func (c *CircuitBreaker) toOpen(reason string) {
	c.state = Open
	c.openedAt = time.Now()
	c.lastOpenReason = reason
	metrics.CircuitBreakerOpenTotal.WithLabelValues(c.upstreamLabel, reason).Inc()
	metrics.CircuitBreakerState.WithLabelValues(c.upstreamLabel).Set(1)
}
