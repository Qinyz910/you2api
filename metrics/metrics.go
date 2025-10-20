package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerOnce sync.Once

	// HTTP请求计数
	RequestCounter *prometheus.CounterVec

	// 上游请求耗时直方图
	UpstreamRequestDuration *prometheus.HistogramVec

	// 上游重试次数
	UpstreamRetriesTotal *prometheus.CounterVec

	// 熔断开启计数
	CircuitBreakerOpenTotal *prometheus.CounterVec

	// 熔断当前状态（0=closed,1=open,2=half_open）
	CircuitBreakerState *prometheus.GaugeVec

	// 客户端断开计数
	ClientDisconnectsTotal *prometheus.CounterVec
)

func Init() {
	registerOnce.Do(func() {
		RequestCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "HTTP请求总数",
			},
			[]string{"method", "endpoint", "status"},
		)

		UpstreamRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "upstream_request_duration_seconds",
				Help:    "上游请求耗时（秒）",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "upstream", "status"},
		)

		UpstreamRetriesTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "upstream_retries_total",
				Help: "上游请求重试总次数",
			},
			[]string{"upstream", "reason"},
		)

		CircuitBreakerOpenTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "circuit_breaker_open_total",
				Help: "熔断器打开次数",
			},
			[]string{"upstream", "reason"},
		)

		CircuitBreakerState = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "circuit_breaker_state",
				Help: "熔断器状态（0=closed,1=open,2=half_open）",
			},
			[]string{"upstream"},
		)

		ClientDisconnectsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "client_disconnects_total",
				Help: "客户端断开连接次数",
			},
			[]string{"endpoint"},
		)

		prometheus.MustRegister(
			RequestCounter,
			UpstreamRequestDuration,
			UpstreamRetriesTotal,
			CircuitBreakerOpenTotal,
			CircuitBreakerState,
			ClientDisconnectsTotal,
		)
	})
}
