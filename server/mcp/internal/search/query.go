package search

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/vault"
)

type Doc struct {
	ID            string
	Title         string
	Kind          string
	Visibility    string
	Summary       string
	Body          string
	Headings      []string
	Tags          []string
	FM            map[string]any
	UpdatedAt     time.Time
	AccessCount   int
	LastAccess    time.Time
	BacklinkCount int
}

type Result struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	PathHint      string             `json:"path_hint"`
	Kind          string             `json:"kind"`
	Visibility    string             `json:"visibility"`
	Summary       string             `json:"summary"`
	MatchedFields []string           `json:"matched_fields"`
	Score         float64            `json:"score"`
	MatchedVia    string             `json:"matched_via"`
	Signals       map[string]float64 `json:"signals"`
	Excerpt       string             `json:"excerpt,omitempty"`
}

func FromNote(n vault.Note, accessCount int, last time.Time, backlinks int) Doc {
	return Doc{
		ID:            n.ID,
		Title:         n.Title,
		Kind:          n.Kind,
		Visibility:    n.Visibility,
		Summary:       n.Summary,
		Body:          n.Body,
		Headings:      n.Headings,
		Tags:          n.Tags,
		FM:            n.FM,
		UpdatedAt:     n.UpdatedAt,
		AccessCount:   accessCount,
		LastAccess:    last,
		BacklinkCount: backlinks,
	}
}

func ClipLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func EmptyResult(query string) map[string]any {
	return map[string]any{
		"results": []any{},
		"message": "未查询到相关内容",
		"suggestions": []string{
			"缩短关键词：去掉修饰词（'的'/'一个'/'关于'）",
			"改用更通用的词：例如 'kubernetes' 替代 'k8s pod 调度'",
			"检查 scope 权限：当前 token scope 是否包含 read:knowledge",
			"检查 visibility：public 内容只能搜到 public 知识",
		},
		"query_echo":       query,
		"executed_signals": []string{"title", "tags", "frontmatter", "section", "fulltext"},
	}
}

func Score(d Doc, tokens []string, weights, intentMul map[string]float64, halfLife float64, now time.Time) (Result, bool) {
	fmJSON := JSONString(d.FM)
	tagsStr := strings.Join(d.Tags, " ")
	headings := strings.Join(d.Headings, "\n")
	titleHit := Hit(d.Title, tokens)
	tagsHit := Hit(tagsStr, tokens)
	fmHit := Hit(fmJSON, tokens)
	sectionHit := Hit(headings, tokens)
	bodyHit := Hit(d.Body, tokens)
	if !titleHit && !tagsHit && !fmHit && !sectionHit && !bodyHit {
		return Result{}, false
	}
	w := func(name string) float64 {
		v := weights[name]
		if m, ok := intentMul[name]; ok && m != 0 {
			v *= m
		}
		return v
	}
	signals := map[string]float64{
		"title":            boolScore(titleHit, w("title")),
		"tags":             boolScore(tagsHit, w("tags")),
		"frontmatter":      boolScore(fmHit, w("frontmatter")),
		"section":          boolScore(sectionHit, w("section")),
		"fulltext":         boolScore(bodyHit, w("fulltext")),
		"wikilink_backref": float64(d.BacklinkCount) * w("wikilink_backref"),
		"access":           Access(d.AccessCount, d.LastAccess, now, halfLife, w("access")),
		"recency":          Recency(d.UpdatedAt, now, w("recency")),
	}
	score := 0.0
	fields := []string{}
	via := []string{}
	for _, name := range []string{"title", "tags", "frontmatter", "section", "fulltext", "wikilink_backref", "access", "recency"} {
		if signals[name] > 0 {
			score += signals[name]
			fields = append(fields, name)
		}
	}
	if bodyHit {
		via = append(via, "fulltext")
	}
	if signals["wikilink_backref"] > 0 {
		via = append(via, "wikilink_backref")
	}
	if titleHit || tagsHit || fmHit || sectionHit {
		via = append(via, "frontmatter")
	}
	return Result{
		ID:            d.ID,
		Title:         d.Title,
		PathHint:      d.ID,
		Kind:          d.Kind,
		Visibility:    d.Visibility,
		Summary:       d.Summary,
		MatchedFields: fields,
		Score:         score,
		MatchedVia:    strings.Join(via, " + "),
		Signals:       signals,
		Excerpt:       excerpt(d.Body, tokens),
	}, true
}

func SortHits(hits []Result, mode string) {
	sort.SliceStable(hits, func(i, j int) bool {
		switch mode {
		case "recency":
			return hits[i].Signals["recency"] > hits[j].Signals["recency"]
		case "access":
			return hits[i].Signals["access"] > hits[j].Signals["access"]
		case "hot":
			ai, aj := hits[i].Signals["access"], hits[j].Signals["access"]
			if ai == aj {
				return hits[i].Score > hits[j].Score
			}
			return ai > aj
		default:
			return hits[i].Score > hits[j].Score
		}
	})
}

func JSONString(v any) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return s
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func HasTag(tags []string, want string) bool {
	if want == "" {
		return true
	}
	low := strings.ToLower(want)
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), low) {
			return true
		}
	}
	return false
}

func boolScore(hit bool, w float64) float64 {
	if !hit {
		return 0
	}
	return w
}

func excerpt(body string, tokens []string) string {
	if body == "" {
		return ""
	}
	low := strings.ToLower(body)
	idx := -1
	for _, tk := range tokens {
		if tk == "" {
			continue
		}
		if i := strings.Index(low, tk); i >= 0 && (idx < 0 || i < idx) {
			idx = i
		}
	}
	if idx < 0 {
		if len(body) > 180 {
			return body[:180] + "…"
		}
		return body
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + 140
	if end > len(body) {
		end = len(body)
	}
	out := body[start:end]
	if start > 0 {
		out = "…" + out
	}
	if end < len(body) {
		out += "…"
	}
	return out
}
