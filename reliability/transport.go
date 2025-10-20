package reliability

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	defaultClient     *http.Client
	defaultClientOnce sync.Once
)

// GetHTTPClient 返回带有合理连接池与超时配置的共享HTTP客户端
func GetHTTPClient() *http.Client {
	defaultClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   getEnvDuration("UPSTREAM_DIAL_TIMEOUT_MS", 5000),
				KeepAlive: getEnvDuration("UPSTREAM_DIAL_KEEPALIVE_MS", 30000),
			}).DialContext,
			MaxIdleConns:          getEnvInt("UPSTREAM_MAX_IDLE_CONNS", 100),
			MaxIdleConnsPerHost:   getEnvInt("UPSTREAM_MAX_IDLE_CONNS_PER_HOST", 100),
			MaxConnsPerHost:       getEnvInt("UPSTREAM_MAX_CONNS_PER_HOST", 0),
			IdleConnTimeout:       getEnvDuration("UPSTREAM_IDLE_CONN_TIMEOUT_MS", 90000),
			TLSHandshakeTimeout:   getEnvDuration("UPSTREAM_TLS_HANDSHAKE_TIMEOUT_MS", 5000),
			ExpectContinueTimeout: getEnvDuration("UPSTREAM_EXPECT_CONTINUE_TIMEOUT_MS", 1000),
			ResponseHeaderTimeout: getEnvDuration("UPSTREAM_RESPONSE_HEADER_TIMEOUT_MS", 15000),
		}

		defaultClient = &http.Client{
			Transport: transport,
			// 端到端超时通过Context控制，以免影响SSE流
		}
	})
	return defaultClient
}

// GetNonStreamTimeout 端到端非流式请求超时
func GetNonStreamTimeout() time.Duration {
	return getEnvDuration("NONSTREAM_TIMEOUT_MS", 60000)
}

// GetStreamTimeout SSE流式请求超时，防止永不结束
func GetStreamTimeout() time.Duration {
	return getEnvDuration("STREAM_TIMEOUT_MS", 120000)
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvDuration(key string, defMS int) time.Duration {
	return time.Duration(getEnvInt(key, defMS)) * time.Millisecond
}
