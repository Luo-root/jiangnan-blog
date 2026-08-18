// Package inbox 管理独立待办（todo）。
//
// 状态机: pending → reviewing → done | abandoned
// 生命周期: done/abandoned 保留 7 天自动删除。
//
// 每条待办是一个 .md 文件，YAML frontmatter + Markdown 正文。
package inbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Status is the inbox item state.
type Status string

const (
	StatusPending   Status = "pending"
	StatusReviewing Status = "reviewing"
	StatusDone      Status = "done"
	StatusAbandoned Status = "abandoned"
)

func validStatus(s Status) bool {
	switch s {
	case StatusPending, StatusReviewing, StatusDone, StatusAbandoned:
		return true
	}
	return false
}

// Item 是一条待办。
type Item struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Status    Status    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Content   string    `json:"content"` // Markdown 正文（不含 frontmatter）
	Location  string    `json:"location"`
}

// Summary 是 list 返回的摘要，不含正文。
type Summary struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Summary     string    `json:"summary"` // 兼容旧客户端：优先标题，其次描述
	Status      Status    `json:"status"`
}

// Store 管理 inbox 文件的读写。
type Store struct {
	dir  string
	mu   sync.Mutex
	stop chan struct{}
	done chan struct{}
}

// New 创建 Store 并启动后台 7 天清理 goroutine。
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, stop: make(chan struct{}), done: make(chan struct{})}
	go s.cleanupLoop()
	return s, nil
}

// Close 停止后台清理。
func (s *Store) Close() {
	close(s.stop)
	<-s.done
}

// Append 新建一条 pending 待办，返回 id。
func (s *Store) Append(createdBy string, content string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	// 文件名带日期+时间，毫秒精度避免同一秒内多条互相覆盖。
	id := now.Format("2006-01-02T15-04-05.000")
	fname := id + ".md"
	path := filepath.Join(s.dir, fname)

	item := Item{
		ID:        id,
		Kind:      "inbox_todo",
		Status:    StatusPending,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
		Content:   content,
		Location:  path,
	}

	if err := writeItem(path, item); err != nil {
		return "", err
	}
	return id, nil
}

// Update 编辑内容或改变状态。
func (s *Store) Update(id string, status Status, content string) error {
	if status != "" && !validStatus(status) {
		return fmt.Errorf("invalid status: %s", status)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	fname := id + ".md"
	path := filepath.Join(s.dir, fname)
	item, err := readItem(path)
	if err != nil {
		return err
	}

	if status != "" {
		item.Status = status
	}
	if content != "" {
		item.Content = content
	}
	item.UpdatedAt = time.Now()

	return writeItem(path, *item)
}

// List 返回所有待办摘要，按创建时间倒序。
func (s *Store) List() ([]Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(s.dir, "*.md"))
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0)
	for _, p := range files {
		item, err := readItem(p)
		if err != nil {
			continue // 跳过损坏的文件
		}
		out = append(out, Summary{
			ID:          item.ID,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			CreatedBy:   item.CreatedBy,
			Title:       titleOf(item.Content),
			Description: descriptionOf(item.Content),
			Summary:     summarise(item.Content),
			Status:      item.Status,
		})
	}
	return out, nil
}

// Get 读取单条完整内容。
func (s *Store) Get(id string) (*Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	return readItem(path)
}

// ---------------------------------------------------------------------------
// 内部
// ---------------------------------------------------------------------------

type frontmatter struct {
	ID        string    `yaml:"id"`
	Kind      string    `yaml:"kind"`
	Status    Status    `yaml:"status"`
	CreatedBy string    `yaml:"created_by"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

func readItem(path string) (*Item, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(b)
	fm, body := splitFrontmatter(text)
	if fm == "" {
		return nil, errors.New("missing frontmatter")
	}

	var f frontmatter
	if err := yaml.Unmarshal([]byte(fm), &f); err != nil {
		return nil, err
	}
	return &Item{
		ID:        f.ID,
		Kind:      f.Kind,
		Status:    f.Status,
		CreatedBy: f.CreatedBy,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
		Content:   strings.TrimSpace(body),
		Location:  path,
	}, nil
}

func writeItem(path string, item Item) error {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", item.ID))
	sb.WriteString(fmt.Sprintf("kind: %s\n", item.Kind))
	sb.WriteString(fmt.Sprintf("status: %s\n", item.Status))
	sb.WriteString(fmt.Sprintf("created_by: %s\n", item.CreatedBy))
	sb.WriteString(fmt.Sprintf("created_at: %s\n", item.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated_at: %s\n", item.UpdatedAt.Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(item.Content)
	sb.WriteString("\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func splitFrontmatter(text string) (fm string, body string) {
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, "---") {
		return "", t
	}
	// 找第二个 ---
	rest := t[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", t
	}
	return strings.TrimSpace(rest[:idx]), strings.TrimSpace(rest[idx+4:])
}

func titleOf(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			return strings.TrimLeft(t, "# ")
		}
	}
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			if len(t) > 80 {
				return t[:80] + "…"
			}
			return t
		}
	}
	return "未命名待办"
}

// descriptionOf 返回标题之后的第一个非空段落；没有则返回空串。
func descriptionOf(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	seenTitle := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "#") {
			seenTitle = true // 标题行不计入描述
			continue
		}
		if !seenTitle {
			// 无 heading 时，第一个非空行被当作标题，之后的行才算描述
			seenTitle = true
			continue
		}
		if len(t) > 180 {
			return t[:180] + "…"
		}
		return t
	}
	return ""
}

func summarise(content string) string {
	if d := descriptionOf(content); d != "" {
		return d
	}
	return titleOf(content)
}

// cleanupLoop 每 1 小时清理一次 done/abandoned 超过 7 天的文件。
func (s *Store) cleanupLoop() {
	defer close(s.done)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(s.dir, "*.md"))
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for _, p := range files {
		item, err := readItem(p)
		if err != nil {
			continue
		}
		if (item.Status == StatusDone || item.Status == StatusAbandoned) && item.UpdatedAt.Before(cutoff) {
			_ = os.Remove(p)
		}
	}
}
