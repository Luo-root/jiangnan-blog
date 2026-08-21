package proposal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/comment"
)

func TestCreateIDAndStatus(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Create(Proposal{
		Target:    Target{Type: "note", Path: "部署溯源/n.md"},
		Operation: Operation{Type: "append"},
		Payload:   Payload{Format: "markdown", Content: "x"},
		CreatedBy: "bot",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusPending {
		t.Fatalf("status = %s", p.Status)
	}
	if p.Kind != "note" {
		t.Fatalf("kind = %s", p.Kind)
	}
	if len(p.ID) < 14 || p.ID[:5] != "prop_" {
		t.Fatalf("id = %s", p.ID)
	}
	got, err := s.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Target.Path != "部署溯源/n.md" {
		t.Fatalf("path = %s", got.Target.Path)
	}
}

func TestStatusMachine(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Create(Proposal{Target: Target{Type: "note", Path: "a.md"}, Operation: Operation{Type: "append"}, Payload: Payload{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateStatus(p.ID, StatusApplied); err == nil {
		t.Fatal("pending cannot go applied")
	}
	if _, err := s.UpdateStatus(p.ID, StatusApproved); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateStatus(p.ID, StatusConflict); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateStatus(p.ID, StatusApproved); err != nil {
		t.Fatal(err)
	}
}

func TestConflictEditAllowed(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.Create(Proposal{Target: Target{Type: "note", Path: "a.md"}, Operation: Operation{Type: "append"}, Payload: Payload{Content: "x"}})
	s.UpdateStatus(p.ID, StatusApproved)
	s.UpdateStatus(p.ID, StatusConflict)
	reason := "retry"
	if _, err := s.Update(p.ID, ProposalPatch{Reason: &reason}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	p, _ := s.Create(Proposal{Target: Target{Type: "note", Path: "a.md"}, Operation: Operation{Type: "append"}, Payload: Payload{Content: "x"}, BaseCommit: "abc"})
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) != 1 {
		t.Fatalf("files = %v", files)
	}
	b, _ := os.ReadFile(files[0])
	if string(b[:3]) != "---" {
		t.Fatalf("want frontmatter, got %s", b)
	}
	got, _ := s.Get(p.ID)
	if got.BaseCommit != "abc" {
		t.Fatalf("base = %s", got.BaseCommit)
	}
}

func TestUpdateTerminalRejectedAndNotFound(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Create(Proposal{Target: Target{Type: "note", Path: "a.md"}, Operation: Operation{Type: "append"}, Payload: Payload{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateStatus(p.ID, StatusRejected); err != nil {
		t.Fatal(err)
	}
	reason := "nope"
	_, err = s.Update(p.ID, ProposalPatch{Reason: &reason})
	if err == nil || !strings.Contains(err.Error(), "terminal or in-flight") {
		t.Fatalf("rejected update: %v", err)
	}
	_, err = s.Get("prop_missing")
	if err == nil || !strings.Contains(err.Error(), "proposal not found") {
		t.Fatalf("get missing: %v", err)
	}
	_, err = s.Update("prop_missing", ProposalPatch{Reason: &reason})
	if err == nil || !strings.Contains(err.Error(), "proposal not found") {
		t.Fatalf("update missing: %v", err)
	}
}

func TestUpdateCommentOnPending(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Create(Proposal{Target: Target{Type: "note", Path: "a.md"}, Operation: Operation{Type: "append"}, Payload: Payload{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := comment.New("agent", "bot", comment.Input{Body: "will retry"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Update(p.ID, ProposalPatch{Comment: &c})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusPending {
		t.Fatalf("status = %s", got.Status)
	}
	if len(got.Comments) != 1 || got.Comments[0].Body != "will retry" {
		t.Fatalf("comments = %+v", got.Comments)
	}
}
