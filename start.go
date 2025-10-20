package main

import (
    "fmt"
    "log"
    "net/http"

    api "you2api/api" // 请替换为您的实际项目名
    config "you2api/config"
    proxy "you2api/proxy"
    metrics "you2api/metrics"

    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    if err := run(); err != nil {
        log.Fatalf("运行错误: %v", err)
    }
}

func run() error {
    // 加载配置
    config, err := config.Load()
    if err != nil {
        return fmt.Errorf("加载配置失败: %w", err)
    }

    // 初始化指标
    metrics.Init()
    http.Handle("/metrics", promhttp.Handler())

    // 如果启用代理
    if config.Proxy.EnableProxy {
        proxy, err := proxy.NewProxy(config.Proxy.ProxyURL, config.Proxy.ProxyTimeoutMS)
        if err != nil {
            return fmt.Errorf("初始化代理失败: %w", err)
        }

        // 注册代理处理器
        http.Handle("/proxy/", http.StripPrefix("/proxy", proxy))
    }

    // 注册API处理器到根路径
    http.HandleFunc("/", api.Handler)

    port := fmt.Sprintf(":%d", config.Port)
    fmt.Printf("Server is running on http://0.0.0.0%s\n", port)

    // 启动服务器
    if err := http.ListenAndServe("0.0.0.0"+port, nil); err != nil {
        return fmt.Errorf("启动服务器失败: %w", err)
    }
    return nil
}
