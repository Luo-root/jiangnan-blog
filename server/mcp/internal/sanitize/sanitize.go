// Package sanitize 做敏感信息检测与脱敏。
package sanitize

import "regexp"

// Patterns 命中即视为敏感内容。
var Patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)authorization:\s*bearer\s+`),
	regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)secret\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)token\s*[:=]\s*\S+`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`(?i)password\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)passwd\s*[:=]\s*\S+`),
	regexp.MustCompile(`\.env`),
	// 公网 IPv4 模式（按策略拦截）
	regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
	// Windows 用户私钥路径
	regexp.MustCompile(`[A-Za-z]:\\Users\\[^\\\s]+`),
	regexp.MustCompile(`\.pem\b`),
}

// FindSensitive 返回命中的敏感模式（去重，最多 8 条）。
func FindSensitive(content string) []string {
	seen := map[string]bool{}
	var out []string
	for _, re := range Patterns {
		if m := re.FindString(content); m != "" {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
		if len(out) >= 8 {
			break
		}
	}
	return out
}

// Redact 把常见敏感值替换为 [REDACTED]。
func Redact(content string) string {
	out := content
	for _, re := range Patterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
