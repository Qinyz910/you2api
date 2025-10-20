package reliability

import (
    "context"
    "errors"
    "fmt"
    "math"
    "math/rand"
    "net"
    "net/http"
    "time"

    metrics "you2api/metrics"
)

// RequestDoer creates a request bound to the provided context and returns it for execution.
// The caller will execute it with the shared client.
type RequestBuilder func(ctx context.Context) (*http.Request, error)

type DoResult struct {
    Resp       *http.Response
    Attempts   int
    LastErr    error
    LastReason string // reason for last retryable failure
}

// DoWithRetry 执行带有指数退避的重试逻辑
func DoWithRetry(ctx context.Context, upstreamLabel string, client *http.Client, build RequestBuilder) (*http.Response, string, error) {
    maxRetries := getEnvInt("UPSTREAM_MAX_RETRIES", 2)
    baseDelay := getEnvDuration("UPSTREAM_RETRY_BASE_DELAY_MS", 200)
    maxDelay := getEnvDuration("UPSTREAM_RETRY_MAX_DELAY_MS", 2000)

    attempt := 0
    var lastErr error
    var lastReason string

    for {
        select {
        case <-ctx.Done():
            if lastErr == nil {
                lastErr = ctx.Err()
            }
            return nil, "context", lastErr
        default:
        }

        req, err := build(ctx)
        if err != nil {
            return nil, "build", err
        }

        start := time.Now()
        resp, err := client.Do(req)
        statusLabel := "error"
        if err == nil {
            statusLabel = fmt.Sprintf("%d", resp.StatusCode)
        }
        metrics.UpstreamRequestDuration.WithLabelValues(req.Method, upstreamLabel, statusLabel).Observe(time.Since(start).Seconds())

        if err == nil && resp.StatusCode < 500 {
            return resp, lastReason, nil
        }

        // 需要重试
        retry, reason := shouldRetry(err, resp)
        if !retry || attempt >= maxRetries {
            if err != nil {
                return nil, reason, err
            }
            return resp, reason, nil
        }

        // 关闭 resp 以免泄漏
        if resp != nil && resp.Body != nil {
            resp.Body.Close()
        }

        attempt++
        lastErr = err
        lastReason = reason
        metrics.UpstreamRetriesTotal.WithLabelValues(upstreamLabel, reason).Inc()

        // 指数退避带抖动
        delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
        if delay > maxDelay {
            delay = maxDelay
        }
        jitter := time.Duration(rand.Int63n(int64(delay / 5))) // 20% 抖动
        select {
        case <-time.After(delay + jitter):
        case <-ctx.Done():
            return nil, "context", ctx.Err()
        }
    }
}

func shouldRetry(err error, resp *http.Response) (bool, string) {
    if err != nil {
        // 如果是上下文取消，说明客户端断开或超时，不能重试
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            return false, "context"
        }
        var netErr net.Error
        if errors.As(err, &netErr) {
            if netErr.Timeout() {
                return true, "timeout"
            }
            // 临时网络错误
            return true, "network"
        }
        // 其他未知错误不重试
        return false, "error"
    }
    if resp != nil {
        if resp.StatusCode == http.StatusTooManyRequests {
            return true, "429"
        }
        if resp.StatusCode >= 500 {
            if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
                return true, fmt.Sprintf("%d", resp.StatusCode)
            }
            return true, "5xx"
        }
    }
    return false, ""
}
