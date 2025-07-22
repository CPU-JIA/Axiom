package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
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

// IsValidPassword 验证密码强度
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	
	// 至少包含一个数字
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	// 至少包含一个字母
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	
	return hasNumber && hasLetter
}

// SanitizeString 清理字符串，移除不安全字符
func SanitizeString(s string) string {
	// 移除HTML标签和特殊字符
	reg := regexp.MustCompile(`<.*?>`)
	return reg.ReplaceAllString(s, "")
}