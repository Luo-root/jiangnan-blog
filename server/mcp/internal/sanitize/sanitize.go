// Package sanitize 做敏感模式检测。命中只 warning，不拒绝、读出不打码。
// 运行时权威 = config.yaml schema.sensitive_patterns；空列表 = 关闭。
package sanitize

import "regexp"

// Find 返回命中的原文片段（去重，最多 8 条）。patterns 空 = 不检测。
func Find(patterns []string, texts ...string) []string {
	if len(patterns) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			continue
		}
		for _, text := range texts {
			if m := re.FindString(text); m != "" && !seen[m] {
				seen[m] = true
				out = append(out, m)
				if len(out) >= 8 {
					return out
				}
			}
		}
	}
	return out
}
