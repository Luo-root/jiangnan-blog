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
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusApplied  Status = "applied"
	StatusRejected Status = "rejected"
	StatusConflict Status = "conflict"
)

type Proposal struct {
	ID     string `json:"id" yaml:"id"`
	Kind   string `json:"kind" yaml:"kind"`
	Status Status `json:"status" yaml:"status"`
	Reason string `json:"reason" yaml:"reason"`

	CreatedBy  string    `json:"created_by" yaml:"created_by"`
	CreatedAt  time.Time `json:"created_at" yaml:"created_at"`
	BaseCommit string    `json:"base_commit,omitempty" yaml:"base_commit,omitempty"`

	Target     Target     `json:"target" yaml:"target"`
	Operation  Operation  `json:"operation" yaml:"operation"`
	Payload    Payload    `json:"payload" yaml:"payload"`
	Risk       Risk       `json:"risk,omitempty" yaml:"risk,omitempty"`
	Validation Validation `json:"validation,omitempty" yaml:"validation,omitempty"`

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

type Risk struct {
	Level   string   `json:"level,omitempty" yaml:"level,omitempty"`
	Reasons []string `json:"reasons,omitempty" yaml:"reasons,omitempty"`
}

type Validation struct {
	OK       bool     `json:"ok" yaml:"ok"`
	Checks   []string `json:"checks,omitempty" yaml:"checks,omitempty"`
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

type Receipt struct {
	Status          Status           `json:"status" yaml:"status"`
	AppliedAt       time.Time        `json:"applied_at,omitempty" yaml:"applied_at,omitempty"`
	Commit          string           `json:"commit,omitempty" yaml:"commit,omitempty"`
	ContentSHA      string           `json:"content_sha256,omitempty" yaml:"content_sha256,omitempty"`
	BaseCommit      string           `json:"base_commit,omitempty" yaml:"base_commit,omitempty"`
	MergeStrategy   string           `json:"merge_strategy,omitempty" yaml:"merge_strategy,omitempty"`
	Replayed        bool             `json:"replayed" yaml:"replayed"`
	ConflictRegions []ConflictRegion `json:"conflict_regions,omitempty" yaml:"conflict_regions,omitempty"`
}

type ConflictRegion struct {
	Excerpt string `json:"excerpt,omitempty" yaml:"excerpt,omitempty"`
}

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

func (s *Store) Create(p Proposal) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	p.ID = nextIDLocked(s.dir, now)
	p.Status = StatusPending
	p.CreatedAt = now
	if p.Kind == "" {
		p.Kind = p.Target.Type
	}

	path := filepath.Join(s.dir, p.ID+".md")
	if err := writeProposal(path, p); err != nil {
		return nil, err
	}
	return &p, nil
}

func nextIDLocked(dir string, now time.Time) string {
	prefix := "prop_" + now.Format("20060102") + "_"
	max := 0
	files, _ := filepath.Glob(filepath.Join(dir, prefix+"*.md"))
	for _, f := range files {
		nstr := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(f), ".md"), prefix)
		n, err := strconv.Atoi(nstr)
		if err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s%03d", prefix, max+1)
}

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

func (s *Store) Get(id string) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readProposal(filepath.Join(s.dir, id+".md"))
}

func (s *Store) Update(id string, patch ProposalPatch) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	p, err := readProposal(path)
	if err != nil {
		return nil, err
	}
	if p.Status != StatusPending && p.Status != StatusConflict {
		return nil, fmt.Errorf("proposal %s is %s, only pending or conflict proposal can be edited", id, p.Status)
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
	if patch.BaseCommit != nil {
		p.BaseCommit = *patch.BaseCommit
	}

	if err := writeProposal(path, *p); err != nil {
		return nil, err
	}
	return p, nil
}

type ProposalPatch struct {
	Reason     *string    `json:"reason,omitempty"`
	Target     *Target    `json:"target,omitempty"`
	Operation  *Operation `json:"operation,omitempty"`
	Payload    *Payload   `json:"payload,omitempty"`
	BaseCommit *string    `json:"base_commit,omitempty"`
}

func (s *Store) UpdateStatus(id string, status Status) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, id+".md")
	p, err := readProposal(path)
	if err != nil {
		return nil, err
	}

	switch status {
	case StatusApproved:
		if p.Status != StatusPending && p.Status != StatusConflict {
			return nil, fmt.Errorf("proposal %s is %s, only pending or conflict proposal can be approved", id, p.Status)
		}
	case StatusRejected:
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

func (s *Store) Save(p *Proposal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeProposal(filepath.Join(s.dir, p.ID+".md"), *p)
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
