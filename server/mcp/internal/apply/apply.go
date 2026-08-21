// Package apply 负责 Proposal 批准后的写入（SCHEMA / 设计 §17.3）。
//
// A 构造完整 ours（在 base_commit 文件上施加 operation）
// B 3-way（HEAD=base 则直接落盘；payload 片段不当 ours）
// C 落盘 + git commit
// D reindex / rebuild 是调用方副作用，失败仍 applied
package apply

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/proposal"
)

type Deps struct {
	VaultRoot  string
	GitDir     string
	VisDefault map[string]string
}

func Apply(p *proposal.Proposal, deps Deps) (*proposal.Receipt, error) {
	if p.Receipt != nil && p.Receipt.Status == proposal.StatusApplied {
		r := *p.Receipt
		r.Replayed = true
		return &r, nil
	}

	abs, rel, err := ResolvePath(deps.VaultRoot, p.Target.Path)
	if err != nil {
		return conflictReceipt(p, "", nil), fmt.Errorf("target_path_invalid")
	}

	create := p.Operation.Type == "create_file" || p.Operation.Type == "register_item"
	now := time.Now()

	if create {
		if _, err := os.Stat(abs); err == nil {
			return &proposal.Receipt{
				Status:        proposal.StatusConflict,
				AppliedAt:     now,
				BaseCommit:    p.BaseCommit,
				MergeStrategy: "none",
			}, fmt.Errorf("target_already_exists")
		}
		ours, err := applyOp("", p.Operation, p.Payload.Content)
		if err != nil {
			return conflictReceipt(p, "none", nil), err
		}
		if vis := ContentVisibility(ours, rel, deps.VisDefault); vis == "secret" {
			return conflictReceipt(p, "none", nil), fmt.Errorf("visibility_not_writable")
		}
		if err := writeFile(abs, ours); err != nil {
			return conflictReceipt(p, "none", nil), err
		}
		hash := sha256Hex(ours)
		commit, err := Commit(deps.GitDir, deps.VaultRoot, rel, "workbase: apply "+p.ID)
		if err != nil {
			Restore(deps.GitDir, deps.VaultRoot, rel)
			_ = os.Remove(abs)
			return conflictReceipt(p, "none", nil), err
		}
		return &proposal.Receipt{
			Status:        proposal.StatusApplied,
			AppliedAt:     now,
			Commit:        commit,
			ContentSHA:    hash,
			BaseCommit:    p.BaseCommit,
			MergeStrategy: "none",
		}, nil
	}

	if vis := FileVisibility(abs, rel, deps.VisDefault); vis == "secret" {
		return conflictReceipt(p, "", nil), fmt.Errorf("visibility_not_writable")
	}

	baseText, err := fileAtBase(deps, rel, abs, p.BaseCommit)
	if err != nil {
		return conflictReceipt(p, "", nil), err
	}
	ours, err := applyOp(baseText, p.Operation, p.Payload.Content)
	if err != nil {
		return conflictReceipt(p, "", nil), err
	}

	head := Head(deps.GitDir)
	strategy := "none"
	final := ours
	if deps.GitDir != "" && p.BaseCommit != "" && head != "" && head != p.BaseCommit {
		other, err := Show(deps.GitDir, head, rel)
		if err != nil {
			otherBytes, readErr := os.ReadFile(abs)
			if readErr != nil {
				return conflictReceipt(p, "three_way", nil), err
			}
			other = string(otherBytes)
		}
		merged, conflict, mergeErr := MergeFile(ours, baseText, other)
		if mergeErr != nil {
			return conflictReceipt(p, "three_way", nil), mergeErr
		}
		if conflict || frontmatterConflict(merged) {
			regions := parseConflicts(merged)
			return &proposal.Receipt{
				Status:          proposal.StatusConflict,
				AppliedAt:       now,
				BaseCommit:      p.BaseCommit,
				MergeStrategy:   "three_way",
				ConflictRegions: regions,
			}, fmt.Errorf("merge_conflict")
		}
		final = merged
		strategy = "three_way"
	}

	if vis := ContentVisibility(final, rel, deps.VisDefault); vis == "secret" {
		return conflictReceipt(p, strategy, nil), fmt.Errorf("visibility_not_writable")
	}
	if err := writeFile(abs, final); err != nil {
		return conflictReceipt(p, strategy, nil), err
	}
	hash := sha256Hex(final)
	commit, err := Commit(deps.GitDir, deps.VaultRoot, rel, "workbase: apply "+p.ID)
	if err != nil {
		Restore(deps.GitDir, deps.VaultRoot, rel)
		return conflictReceipt(p, strategy, nil), err
	}
	return &proposal.Receipt{
		Status:        proposal.StatusApplied,
		AppliedAt:     now,
		Commit:        commit,
		ContentSHA:    hash,
		BaseCommit:    p.BaseCommit,
		MergeStrategy: strategy,
	}, nil
}

func fileAtBase(deps Deps, rel, abs, base string) (string, error) {
	if deps.GitDir != "" && base != "" {
		text, err := Show(deps.GitDir, base, rel)
		if err == nil {
			return text, nil
		}
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("target_not_found")
	}
	return string(b), nil
}

func writeFile(abs, content string) error {
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	if content != "" && content[len(content)-1] != '\n' {
		content += "\n"
	}
	return os.WriteFile(abs, []byte(content), 0644)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func conflictReceipt(p *proposal.Proposal, strategy string, regions []proposal.ConflictRegion) *proposal.Receipt {
	return &proposal.Receipt{
		Status:          proposal.StatusConflict,
		AppliedAt:       time.Now(),
		BaseCommit:      p.BaseCommit,
		MergeStrategy:   strategy,
		ConflictRegions: regions,
	}
}
