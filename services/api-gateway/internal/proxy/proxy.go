package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"api-gateway/internal/config"
	"api-gateway/pkg/logger"
	"api-gateway/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ServiceProxy 服务代理
type ServiceProxy struct {
	services map[string]config.ServiceConfig
	client   *http.Client
	logger   logger.Logger
}

// NewServiceProxy 创建服务代理
func NewServiceProxy(cfg *config.Config, logger logger.Logger) *ServiceProxy {
	return &ServiceProxy{
		services: cfg.Services,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout.Handler) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// ProxyRequest 代理请求到后端服务
func (p *ServiceProxy) ProxyRequest(c *gin.Context) {
	// 提取服务名
	serviceName := p.extractServiceName(c.Request.URL.Path)
	if serviceName == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "SERVICE_NOT_FOUND",
				"message": "Service not found in path",
			},
		})
		return
	}

	// 获取服务配置
	serviceConfig, exists := p.services[serviceName+"-service"]
	if !exists {
		p.logger.Warn("Service not configured", "service", serviceName)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": fmt.Sprintf("Service '%s' is not available", serviceName),
			},
		})
		return
	}

	// 构建目标URL
	targetURL := p.buildTargetURL(serviceConfig, c.Request.URL)
	
	// 创建代理请求
	req, err := p.createProxyRequest(c, targetURL)
	if err != nil {
		p.logger.Error("Failed to create proxy request", "error", err, "target", targetURL)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PROXY_REQUEST_FAILED",
				"message": "Failed to create proxy request",
			},
		})
		return
	}

	// 执行请求
	resp, err := p.executeRequest(req, serviceConfig)
	if err != nil {
		p.logger.Error("Proxy request failed", 
			"error", err, 
			"service", serviceName, 
			"target", targetURL,
			"method", c.Request.Method,
		)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"code":    "GATEWAY_ERROR",
				"message": "Failed to connect to backend service",
			},
		})
		return
	}
	defer resp.Body.Close()

	// 复制响应
	p.copyResponse(c, resp)

	// 记录日志
	p.logger.Info("Request proxied successfully",
		"service", serviceName,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", resp.StatusCode,
		"target", targetURL,
	)
}

// extractServiceName 从路径中提取服务名
func (p *ServiceProxy) extractServiceName(path string) string {
	// /api/v1/auth/login -> auth
	// /api/v1/tenants/123 -> tenants
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		service := parts[2]
		// 处理特殊路由映射
		switch service {
		case "auth":
			return "iam"
		case "tenants":
			return "tenant"
		case "projects":
			return "project"
		case "git":
			return "git-gateway"
		case "cicd", "pipelines":
			return "cicd"
		case "notifications":
			return "notification"
		case "kb", "wiki":
			return "kb"
		default:
			return service
		}
	}
	return ""
}

// buildTargetURL 构建目标URL
func (p *ServiceProxy) buildTargetURL(serviceConfig config.ServiceConfig, requestURL *url.URL) string {
	targetURL := strings.TrimSuffix(serviceConfig.URL, "/")
	path := utils.SanitizePath(requestURL.Path)
	
	// 构建完整URL
	fullURL := targetURL + path
	
	// 添加查询参数
	if requestURL.RawQuery != "" {
		fullURL += "?" + requestURL.RawQuery
	}
	
	return fullURL
}

// createProxyRequest 创建代理请求
func (p *ServiceProxy) createProxyRequest(c *gin.Context, targetURL string) (*http.Request, error) {
	// 读取请求体
	var body io.Reader
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bodyBytes)
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// 创建请求
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, body)
	if err != nil {
		return nil, err
	}

	// 复制请求头
	p.copyRequestHeaders(c, req)

	// 添加代理信息头
	p.addProxyHeaders(c, req)

	return req, nil
}

// executeRequest 执行请求（支持重试）
func (p *ServiceProxy) executeRequest(req *http.Request, serviceConfig config.ServiceConfig) (*http.Response, error) {
	var lastErr error
	maxRetries := serviceConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for i := 0; i < maxRetries; i++ {
		// 创建超时上下文
		ctx, cancel := context.WithTimeout(req.Context(), time.Duration(serviceConfig.Timeout)*time.Second)
		reqWithTimeout := req.WithContext(ctx)

		resp, err := p.client.Do(reqWithTimeout)
		cancel()

		if err == nil {
			return resp, nil
		}

		lastErr = err
		p.logger.Warn("Request attempt failed", 
			"attempt", i+1, 
			"max_retries", maxRetries, 
			"error", err,
			"target", req.URL.String(),
		)

		// 最后一次尝试不需要等待
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("all %d attempts failed, last error: %w", maxRetries, lastErr)
}

// copyRequestHeaders 复制请求头
func (p *ServiceProxy) copyRequestHeaders(c *gin.Context, req *http.Request) {
	// 需要跳过的头部
	skipHeaders := map[string]bool{
		"connection":       true,
		"upgrade":          true,
		"proxy-connection": true,
		"te":               true,
		"trailer":          true,
		"transfer-encoding": true,
	}

	for key, values := range c.Request.Header {
		if !skipHeaders[strings.ToLower(key)] {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
}

// addProxyHeaders 添加代理信息头
func (p *ServiceProxy) addProxyHeaders(c *gin.Context, req *http.Request) {
	// 添加X-Forwarded-For头
	clientIP := c.ClientIP()
	if existingXFF := req.Header.Get("X-Forwarded-For"); existingXFF != "" {
		req.Header.Set("X-Forwarded-For", existingXFF+", "+clientIP)
	} else {
		req.Header.Set("X-Forwarded-For", clientIP)
	}

	// 添加X-Forwarded-Proto头
	if c.Request.TLS != nil {
		req.Header.Set("X-Forwarded-Proto", "https")
	} else {
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	// 添加X-Forwarded-Host头
	req.Header.Set("X-Forwarded-Host", c.Request.Host)

	// 添加X-Real-IP头
	req.Header.Set("X-Real-IP", clientIP)

	// 添加请求ID（如果存在）
	if requestID := c.GetString("request_id"); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
}

// copyResponse 复制响应
func (p *ServiceProxy) copyResponse(c *gin.Context, resp *http.Response) {
	// 复制状态码
	c.Status(resp.StatusCode)

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 复制响应体
	_, err := io.Copy(c.Writer, resp.Body)
	if err != nil {
		p.logger.Error("Failed to copy response body", "error", err)
	}
}

// HealthCheck 检查后端服务健康状态
func (p *ServiceProxy) HealthCheck(serviceName string) error {
	serviceConfig, exists := p.services[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not configured", serviceName)
	}

	healthURL := strings.TrimSuffix(serviceConfig.URL, "/") + serviceConfig.HealthEndpoint
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(serviceConfig.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}