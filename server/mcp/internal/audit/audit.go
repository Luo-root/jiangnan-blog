// Package audit 记录工具调用审计（op + 时间 + scope + 目标/哈希）。
//
// v0.1 最小实现：JSONL 追加写 + 内存缓存最近条目。
// detail 模式返回目标 id；hashed 模式返回内容哈希，都不含正文/查询原文。
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry 是一条审计记录。
type Entry struct {
	Time        time.Time `json:"time"`
	Op          string    `json:"op"`                     // 工具名，如 knowledge.get
	Scope       string    `json:"scope"`                  // 所需 scope
	ClientID    string    `json:"client_id,omitempty"`    // 调用方 id
	TargetID    string    `json:"target_id,omitempty"`    // 目标 id（detail 模式返回）
	ContentHash string    `json:"content_hash,omitempty"` // 内容 SHA-256（hashed 模式返回）
}

// Store 持有一条审计文件路径 + 内存缓存。
type Store struct {
	mu      sync.Mutex
	path    string
	entries []Entry
	max     int
}

// New 打开（或创建）审计文件并载入最近条目。
func New(path string, max int) *Store {
	if max <= 0 {
		max = 2000
	}
	s := &Store{path: path, max: max}
	s.load()
	return s
}

// Append 记录一条审计，追加写盘。
func (s *Store) Append(op, scope, clientID, targetID, rawContent string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := Entry{
		Time:     time.Now(),
		Op:       op,
		Scope:    scope,
		ClientID: clientID,
		TargetID: targetID,
	}
	if rawContent != "" {
		e.ContentHash = hash(rawContent)
	}

	s.entries = append(s.entries, e)
	if len(s.entries) > s.max {
		s.entries = s.entries[len(s.entries)-s.max:]
	}

	if s.path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(s.path), 0755)
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(e)
}

// List 返回最近 limit 条，按时间倒序。
// mode=detail 返回 TargetID；mode=hashed 返回 ContentHash；其余等价 detail。
func (s *Store) List(mode string, limit int) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if mode == "hashed" {
			e.TargetID = ""
		} else {
			e.ContentHash = ""
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Store) load() {
	if s.path == "" {
		return
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	// JSONL：逐行解析，只保留最近 max 条。
	lines := splitLines(string(b))
	for _, l := range lines {
		var e Entry
		if json.Unmarshal([]byte(l), &e) != nil {
			continue
		}
		s.entries = append(s.entries, e)
	}
	if len(s.entries) > s.max {
		s.entries = s.entries[len(s.entries)-s.max:]
	}
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if l := s[start:i]; len(l) > 0 {
				out = append(out, l)
			}
			start = i + 1
		}
	}
	if l := s[start:]; len(l) > 0 {
		out = append(out, l)
	}
	return out
}
