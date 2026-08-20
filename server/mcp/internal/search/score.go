package search

import (
	"math"
	"strings"
	"time"
	"unicode"
)

func Tokenize(q string) []string {
	var out []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, strings.ToLower(buf.String()))
		buf.Reset()
	}
	for _, r := range q {
		if unicode.IsSpace(r) || strings.ContainsRune("，。.!！?、；;:：", r) {
			flush()
			continue
		}
		buf.WriteRune(r)
	}
	flush()
	return out
}

func Hit(s string, tokens []string) bool {
	if s == "" || len(tokens) == 0 {
		return false
	}
	low := strings.ToLower(s)
	for _, tk := range tokens {
		if tk != "" && strings.Contains(low, tk) {
			return true
		}
	}
	return false
}

func Recency(updated, now time.Time, weight float64) float64 {
	if updated.IsZero() || weight == 0 {
		return 0
	}
	days := now.Sub(updated).Hours() / 24
	if days < 0 {
		days = 0
	}
	return weight * math.Exp(-days/30)
}

func Access(count int, last, now time.Time, halfLifeDays, weight float64) float64 {
	if count <= 0 || last.IsZero() || weight == 0 {
		return 0
	}
	if halfLifeDays <= 0 {
		halfLifeDays = 7
	}
	days := now.Sub(last).Hours() / 24
	if days < 0 {
		days = 0
	}
	return weight * float64(count) * math.Exp(-days/halfLifeDays)
}
