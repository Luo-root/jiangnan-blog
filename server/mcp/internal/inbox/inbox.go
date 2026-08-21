// Package inbox 管理独立待办（todo）。
//
// 状态机: pending → reviewing → done | abandoned
// 生命周期: done/abandoned 超过 retention_days 自动删除。
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

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/comment"
	"gopkg.in/yaml.v3"
)

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

func canTransit(from, to Status) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusPending:
		return to == StatusReviewing || to == StatusDone || to == StatusAbandoned
	case StatusReviewing:
		return to == StatusDone || to == StatusAbandoned
	default:
		return false
	}
}

type Item struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Status    Status            `json:"status"`
	CreatedBy string            `json:"created_by"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Title     string            `json:"title,omitempty"`
	Content   string            `json:"content"`
	Tags      []string          `json:"tags,omitempty"`
	Comments  []comment.Comment `json:"comments"`
	Location  string            `json:"location"`
}

type Summary struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedBy    string    `json:"created_by"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Summary      string    `json:"summary"`
	Status       Status    `json:"status"`
	CommentCount int       `json:"comment_count"`
}

type Store struct {
	dir           string
	retentionDays int
	mu            sync.Mutex
	stop          chan struct{}
	done          chan struct{}
}

func New(dir string, retentionDays int) (*Store, error) {
	if retentionDays <= 0 {
		retentionDays = 7
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, retentionDays: retentionDays, stop: make(chan struct{}), done: make(chan struct{})}
	go s.cleanupLoop()
	return s, nil
}

func (s *Store) Close() {
	close(s.stop)
	<-s.done
}

func (s *Store) Append(createdBy, content, title string, tags []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	id := "inbox_" + now.Format("20060102_150405") + fmt.Sprintf("_%03d", now.Nanosecond()/1e6)
	path := filepath.Join(s.dir, id+".md")
	item := Item{
		ID:        id,
		Kind:      "inbox_todo",
		Status:    StatusPending,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
		Title:     title,
		Content:   content,
		Tags:      tags,
		Comments:  []comment.Comment{},
		Location:  path,
	}
	if err := writeItem(path, item); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Update(id string, status Status, content, title string, tags []string, cmt *comment.Comment) error {
	if status != "" && !validStatus(status) {
		return fmt.Errorf("invalid status: %s", status)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	item, err := readItem(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("inbox not found: %s", id)
		}
		return err
	}
	if status != "" {
		if !canTransit(item.Status, status) {
			return fmt.Errorf("invalid status transition: %s → %s", item.Status, status)
		}
		item.Status = status
	}
	if content != "" {
		item.Content = content
	}
	if title != "" {
		item.Title = title
	}
	if tags != nil {
		item.Tags = tags
	}
	if cmt != nil {
		item.Comments = append(comment.Slice(item.Comments), *cmt)
	}
	item.UpdatedAt = time.Now()
	return writeItem(path, *item)
}

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
			continue
		}
		title := item.Title
		if title == "" {
			title = titleOf(item.Content)
		}
		out = append(out, Summary{
			ID:           item.ID,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
			CreatedBy:    item.CreatedBy,
			Title:        title,
			Description:  descriptionOf(item.Content),
			Summary:      summarise(item.Content),
			Status:       item.Status,
			CommentCount: len(item.Comments),
		})
	}
	return out, nil
}

func (s *Store) Get(id string) (*Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := readItem(filepath.Join(s.dir, id+".md"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("inbox not found: %s", id)
		}
		return nil, err
	}
	if item.Title == "" {
		item.Title = titleOf(item.Content)
	}
	item.Comments = comment.Slice(item.Comments)
	return item, nil
}

type frontmatter struct {
	ID        string            `yaml:"id"`
	Kind      string            `yaml:"kind"`
	Status    Status            `yaml:"status"`
	CreatedBy string            `yaml:"created_by"`
	CreatedAt time.Time         `yaml:"created_at"`
	UpdatedAt time.Time         `yaml:"updated_at"`
	Title     string            `yaml:"title,omitempty"`
	Tags      []string          `yaml:"tags,omitempty"`
	Comments  []comment.Comment `yaml:"comments,omitempty"`
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
		Title:     f.Title,
		Content:   strings.TrimSpace(body),
		Tags:      f.Tags,
		Comments:  comment.Slice(f.Comments),
		Location:  path,
	}, nil
}

func writeItem(path string, item Item) error {
	fm := frontmatter{
		ID:        item.ID,
		Kind:      item.Kind,
		Status:    item.Status,
		CreatedBy: item.CreatedBy,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
		Title:     item.Title,
		Tags:      item.Tags,
		Comments:  comment.Slice(item.Comments),
	}
	b, err := yaml.Marshal(fm)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(string(b))
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

func descriptionOf(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	seenTitle := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "#") {
			seenTitle = true
			continue
		}
		if !seenTitle {
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

func (s *Store) cleanupLoop() {
	defer close(s.done)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	s.cleanup()
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
	cutoff := time.Now().Add(-time.Duration(s.retentionDays) * 24 * time.Hour)
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
