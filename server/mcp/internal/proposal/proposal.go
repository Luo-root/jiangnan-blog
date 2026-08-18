// Package proposal 管理正式写入请求（Proposal）。
//
// proposal 与 inbox 完全独立：proposal 走 webUI 审批 → apply → commit，
// 有 target + operation + payload，inbox 是待办。
package proposal

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

// Status 是 proposal 状态机。
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusApplied  Status = "applied"
	StatusRejected Status = "rejected"
	StatusConflict Status = "conflict"
)

// Proposal 是一条写入请求。
type Proposal struct {
	ID     string `json:"id" yaml:"id"`
	Kind   string `json:"kind" yaml:"kind"`
	Status Status `json:"status" yaml:"status"`
	Reason string `json:"reason" yaml:"reason"`

	CreatedBy  string    `json:"created_by" yaml:"created_by"`
	CreatedAt  time.Time `json:"created_at" yaml:"created_at"`
	BaseCommit string    `json:"base_commit,omitempty" yaml:"base_commit,omitempty"`

	Target    Target    `json:"target" yaml:"target"`
	Operation Operation `json:"operation" yaml:"operation"`
	Payload   Payload   `json:"payload" yaml:"payload"`

	Receipt *Receipt `json:"receipt,omitempty" yaml:"receipt,omitempty"`
}

type Target struct {
	Type string `json:"type" yaml:"type"`
	ID   string `json:"id,omitempty" yaml:"id,omitempty"`
	Path string `json:"path" yaml:"path"`
}

type Operation struct {
	Type    string `json:"type" yaml:"type"`
	Section string `json:"section,omitempty" yaml:"section,omitempty"`
}

type Payload struct {
	Format  string `json:"format" yaml:"format"`
	Content string `json:"content" yaml:"content"`
}

// Receipt 是 apply 的结果（applied 需要 git commit + hash 校验）。
type Receipt struct {
	Status     Status    `json:"status" yaml:"status"`
	AppliedAt  time.Time `json:"applied_at,omitempty" yaml:"applied_at,omitempty"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	ContentSHA string    `json:"content_sha256,omitempty" yaml:"content_sha256,omitempty"`
	Replayed   bool      `json:"replayed" yaml:"replayed"`
}

// Store 管理 proposal 文件。
type Store struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Create 新建一条 pending proposal。
func (s *Store) Create(p Proposal) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	p.ID = "prop_" + now.Format("20060102_150405")
	p.Status = StatusPending
	p.CreatedAt = now

	path := filepath.Join(s.dir, p.ID+".md")
	if err := writeProposal(path, p); err != nil {
		return nil, err
	}
	return &p, nil
}

// List 返回所有 proposal 摘要。
func (s *Store) List() ([]Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(s.dir, "*.md"))
	if err != nil {
		return nil, err
	}
	out := make([]Proposal, 0)
	for _, p := range files {
		prop, err := readProposal(p)
		if err != nil {
			continue
		}
		out = append(out, *prop)
	}
	return out, nil
}

// Get 读取单条 proposal。
func (s *Store) Get(id string) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readProposal(filepath.Join(s.dir, id+".md"))
}

// Update 修改一条 pending proposal 的可编辑字段（reason / target / operation / payload）。
// 已审批/拒绝/应用的 proposal 不允许再编辑。
func (s *Store) Update(id string, patch ProposalPatch) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	p, err := readProposal(path)
	if err != nil {
		return nil, err
	}
	if p.Status != StatusPending {
		return nil, fmt.Errorf("proposal %s is %s, only pending proposal can be edited", id, p.Status)
	}

	if patch.Reason != nil {
		p.Reason = *patch.Reason
	}
	if patch.Target != nil {
		p.Target = *patch.Target
	}
	if patch.Operation != nil {
		p.Operation = *patch.Operation
	}
	if patch.Payload != nil {
		p.Payload = *patch.Payload
	}

	if err := writeProposal(path, *p); err != nil {
		return nil, err
	}
	return p, nil
}

// ProposalPatch 是需要更新的字段（nil 表示不修改）。
type ProposalPatch struct {
	Reason    *string    `json:"reason,omitempty"`
	Target    *Target    `json:"target,omitempty"`
	Operation *Operation `json:"operation,omitempty"`
	Payload   *Payload   `json:"payload,omitempty"`
}

// UpdateStatus 更新 Proposal 状态。状态转换规则：
//
//	pending → approved / rejected
//	approved → applied / conflict
func (s *Store) UpdateStatus(id string, status Status) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	p, err := readProposal(path)
	if err != nil {
		return nil, err
	}

	switch status {
	case StatusApproved, StatusRejected:
		if p.Status != StatusPending {
			return nil, fmt.Errorf("proposal %s is %s, only pending proposal can be reviewed", id, p.Status)
		}
	case StatusApplied, StatusConflict:
		if p.Status != StatusApproved {
			return nil, fmt.Errorf("proposal %s is %s, only approved proposal can be applied", id, p.Status)
		}
	default:
		return nil, fmt.Errorf("invalid review status: %s", status)
	}

	p.Status = status
	if err := writeProposal(path, *p); err != nil {
		return nil, err
	}
	return p, nil
}

// SetReceipt 写入 receipt 到 proposal 文件。
func (s *Store) SetReceipt(id string, r *Receipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	p, err := readProposal(path)
	if err != nil {
		return err
	}
	p.Receipt = r
	return writeProposal(path, *p)
}

func writeProposal(path string, p Proposal) error {
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(string(b))
	sb.WriteString("---\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func readProposal(path string) (*Proposal, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(b))
	if !strings.HasPrefix(text, "---") {
		return nil, errors.New("missing frontmatter")
	}
	rest := text[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, errors.New("unclosed frontmatter")
	}
	fm := rest[:idx]

	var p Proposal
	if err := yaml.Unmarshal([]byte(fm), &p); err != nil {
		return nil, fmt.Errorf("parse proposal: %w", err)
	}
	return &p, nil
}
