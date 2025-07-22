package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

// GenerateRequestID 生成请求ID
func GenerateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// IsValidURL 验证URL格式
func IsValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

// ExtractServiceName 从路径中提取服务名
func ExtractServiceName(path string) string {
	// /api/v1/auth/login -> auth
	// /api/v1/tenants/123 -> tenants
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		return parts[2]
	}
	return ""
}

// SanitizePath 清理路径
func SanitizePath(path string) string {
	// 移除多余的斜杠
	path = regexp.MustCompile(`/+`).ReplaceAllString(path, "/")
	// 确保以/开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// GetClientIP 从多个HTTP头中获取客户端IP
func GetClientIP(remoteAddr string, headers map[string]string) string {
	// 按优先级检查各种IP头
	ipHeaders := []string{
		"X-Forwarded-For",
		"X-Real-IP", 
		"X-Client-IP",
		"CF-Connecting-IP", // Cloudflare
	}
	
	for _, header := range ipHeaders {
		if ip, exists := headers[header]; exists && ip != "" {
			// X-Forwarded-For 可能包含多个IP，取第一个
			if strings.Contains(ip, ",") {
				ip = strings.TrimSpace(strings.Split(ip, ",")[0])
			}
			if isValidIP(ip) {
				return ip
			}
		}
	}
	
	// 如果没有找到，使用RemoteAddr
	if strings.Contains(remoteAddr, ":") {
		ip := strings.Split(remoteAddr, ":")[0]
		if isValidIP(ip) {
			return ip
		}
	}
	
	return remoteAddr
}

// isValidIP 检查是否是有效的IP地址
func isValidIP(ip string) bool {
	ipRegex := regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$|^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
	return ipRegex.MatchString(ip) && ip != "0.0.0.0" && !strings.HasPrefix(ip, "127.")
}

// NormalizeHeaders 标准化HTTP头
func NormalizeHeaders(headers map[string][]string) map[string]string {
	normalized := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			normalized[strings.ToLower(key)] = values[0]
		}
	}
	return normalized
}