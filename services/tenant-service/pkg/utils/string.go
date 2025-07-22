package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

// HashString 对字符串进行SHA256哈希
func HashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// IsValidEmail 验证邮箱格式
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(email)
}

// SanitizeString 清理字符串，移除不安全字符
func SanitizeString(s string) string {
	// 移除HTML标签和特殊字符
	reg := regexp.MustCompile(`<.*?>`)
	return reg.ReplaceAllString(s, "")
}

// IsValidTenantName 验证租户名称格式
func IsValidTenantName(name string) bool {
	if len(name) < 3 || len(name) > 50 {
		return false
	}
	
	// 只允许字母、数字、连字符和下划线
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", name)
	if !matched {
		return false
	}
	
	// 不能以连字符或下划线开头或结尾
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") ||
		strings.HasSuffix(name, "-") || strings.HasSuffix(name, "_") {
		return false
	}
	
	return true
}

// GenerateSlug 从名称生成slug
func GenerateSlug(name string) string {
	// 转换为小写
	slug := strings.ToLower(name)
	
	// 移除特殊字符，只保留字母数字和空格
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s-]`)
	slug = reg.ReplaceAllString(slug, "")
	
	// 将空格替换为连字符
	slug = regexp.MustCompile(`\s+`).ReplaceAllString(slug, "-")
	
	// 移除多余的连字符
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	
	// 移除开头和结尾的连字符
	slug = strings.Trim(slug, "-")
	
	return slug
}

// IsValidDomainName 验证域名格式（用于自定义域名）
func IsValidDomainName(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	// 域名正则表达式
	pattern := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`
	matched, _ := regexp.MatchString(pattern, domain)
	return matched
}

// ContainsUnicode 检查字符串是否包含Unicode字符
func ContainsUnicode(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}