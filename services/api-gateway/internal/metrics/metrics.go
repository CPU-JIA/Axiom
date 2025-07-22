package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 监控指标
type Metrics struct {
	// HTTP请求相关指标
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge
	
	// 代理相关指标
	ProxyRequestsTotal   *prometheus.CounterVec
	ProxyRequestDuration *prometheus.HistogramVec
	ProxyErrors          *prometheus.CounterVec
	
	// 限流相关指标
	RateLimitHits   *prometheus.CounterVec
	RateLimitBlocks *prometheus.CounterVec
	
	// 认证相关指标
	AuthRequestsTotal *prometheus.CounterVec
	AuthFailures      *prometheus.CounterVec
	
	// 健康检查指标
	HealthChecks *prometheus.GaugeVec
	
	// 系统指标
	ConnectionsActive prometheus.Gauge
}

// NewMetrics 创建监控指标
func NewMetrics() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status", "service"},
		),
		
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "api_gateway_http_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "service"},
		),
		
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),
		
		ProxyRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_proxy_requests_total",
				Help: "Total number of proxy requests",
			},
			[]string{"service", "method", "status"},
		),
		
		ProxyRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "api_gateway_proxy_request_duration_seconds",
				Help: "Duration of proxy requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
		
		ProxyErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_proxy_errors_total",
				Help: "Total number of proxy errors",
			},
			[]string{"service", "error_type"},
		),
		
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"key_type", "service"},
		),
		
		RateLimitBlocks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_blocks_total",
				Help: "Total number of rate limit blocks",
			},
			[]string{"key_type", "service"},
		),
		
		AuthRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_auth_requests_total",
				Help: "Total number of authentication requests",
			},
			[]string{"result"},
		),
		
		AuthFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"reason"},
		),
		
		HealthChecks: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_service_health",
				Help: "Health status of backend services (1=healthy, 0=unhealthy)",
			},
			[]string{"service"},
		),
		
		ConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_connections_active",
				Help: "Current number of active connections",
			},
		),
	}
}

// RecordHTTPRequest 记录HTTP请求指标
func (m *Metrics) RecordHTTPRequest(method, path, service string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	
	m.HTTPRequestsTotal.WithLabelValues(method, path, status, service).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, service).Observe(duration.Seconds())
}

// RecordProxyRequest 记录代理请求指标
func (m *Metrics) RecordProxyRequest(service, method string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	
	m.ProxyRequestsTotal.WithLabelValues(service, method, status).Inc()
	m.ProxyRequestDuration.WithLabelValues(service, method).Observe(duration.Seconds())
}

// RecordProxyError 记录代理错误指标
func (m *Metrics) RecordProxyError(service, errorType string) {
	m.ProxyErrors.WithLabelValues(service, errorType).Inc()
}

// RecordRateLimitHit 记录限流命中指标
func (m *Metrics) RecordRateLimitHit(keyType, service string) {
	m.RateLimitHits.WithLabelValues(keyType, service).Inc()
}

// RecordRateLimitBlock 记录限流阻止指标
func (m *Metrics) RecordRateLimitBlock(keyType, service string) {
	m.RateLimitBlocks.WithLabelValues(keyType, service).Inc()
}

// RecordAuthRequest 记录认证请求指标
func (m *Metrics) RecordAuthRequest(result string) {
	m.AuthRequestsTotal.WithLabelValues(result).Inc()
}

// RecordAuthFailure 记录认证失败指标
func (m *Metrics) RecordAuthFailure(reason string) {
	m.AuthFailures.WithLabelValues(reason).Inc()
}

// SetServiceHealth 设置服务健康状态
func (m *Metrics) SetServiceHealth(service string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.HealthChecks.WithLabelValues(service).Set(value)
}

// IncrementActiveConnections 增加活跃连接数
func (m *Metrics) IncrementActiveConnections() {
	m.ConnectionsActive.Inc()
}

// DecrementActiveConnections 减少活跃连接数
func (m *Metrics) DecrementActiveConnections() {
	m.ConnectionsActive.Dec()
}

// IncrementInFlightRequests 增加进行中的请求数
func (m *Metrics) IncrementInFlightRequests() {
	m.HTTPRequestsInFlight.Inc()
}

// DecrementInFlightRequests 减少进行中的请求数
func (m *Metrics) DecrementInFlightRequests() {
	m.HTTPRequestsInFlight.Dec()
}