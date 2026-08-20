package index

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRebuildSQLiteAndIDs(t *testing.T) {
	vault := t.TempDir()
	mustWrite := func(rel, body string) {
		p := filepath.Join(vault, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("文章/hello.md", "---\nsummary: hi\n---\n# Hello kubernetes\n")
	mustWrite("项目/pulse.md", "---\nname: Pulse\nsummary: s\n---\n# Pulse\n")
	mustWrite("友链/x.md", "---\nname: x\nurl: https://x.test\n---\n")
	mustWrite("Workbase/skills/lint.md", "---\nid: markdown-lint\nkind: skill\nname: Lint\nsummary: s\n---\n# Lint\n")

	db := filepath.Join(t.TempDir(), "notes.sqlite")
	s, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Rebuild(vault, []string{".obsidian", ".trash"}, map[string]string{
		"文章": "public", "项目": "public", "default": "private",
		"Workbase/skills": "private",
	}); err != nil {
		t.Fatal(err)
	}
	n := s.NoteByID("文章/hello.md")
	if n == nil || n.Kind != "article" {
		t.Fatalf("article = %+v", n)
	}
	if s.NoteByID(`文章\hello.md`) == nil {
		t.Fatal("backslash query should normalize")
	}
	if s.NoteByID("文章/hello") != nil {
		t.Fatal("id without .md should not match")
	}
	if s.NoteByID("友链/x.md") != nil {
		t.Fatal("friends indexed")
	}
	notesN, projectsN, skillsN, _, _ := s.Stats()
	if notesN != 3 || projectsN != 1 || skillsN != 1 {
		t.Fatalf("stats notes=%d projects=%d skills=%d", notesN, projectsN, skillsN)
	}
	if s.Hit("文章/hello.md") != 1 {
		t.Fatal("hit")
	}
	if s.Count("文章/hello.md") != 1 {
		t.Fatal("count")
	}
	if err := s.Rebuild(vault, []string{".obsidian", ".trash"}, map[string]string{"文章": "public", "项目": "public", "default": "private", "Workbase/skills": "private"}); err != nil {
		t.Fatal(err)
	}
	if s.Count("文章/hello.md") != 1 {
		t.Fatal("access should survive rebuild")
	}
}

func TestHotScoreOrderAndMinScore(t *testing.T) {
	vaultDir := t.TempDir()
	must := func(rel, body string) {
		p := filepath.Join(vaultDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	must("文章/fresh.md", "# Fresh\n")
	must("文章/stale.md", "# Stale\n")
	must("文章/cold.md", "# Cold\n")
	s, err := Open(filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Rebuild(vaultDir, nil, map[string]string{"文章": "public", "default": "private"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		s.Hit("文章/fresh.md")
		s.Hit("文章/stale.md")
	}
	s.mu.Lock()
	s.lastAcc["文章/stale.md"] = time.Now().Add(-7 * 24 * time.Hour)
	s.lastAcc["文章/cold.md"] = time.Now().Add(-70 * 24 * time.Hour)
	s.access["文章/cold.md"] = 1
	s.mu.Unlock()

	hot := s.Hot(7, 0.001)
	if len(hot) < 2 {
		t.Fatalf("hot = %+v", hot)
	}
	if hot[0].ResourceID != "文章/fresh.md" {
		t.Fatalf("want fresh first, got %+v", hot)
	}
	if math.Abs(hot[1].Score-10/math.E) > 0.05 {
		t.Fatalf("stale score = %v want ~%v", hot[1].Score, 10/math.E)
	}
	for _, h := range hot {
		if h.ResourceID == "文章/cold.md" {
			t.Fatalf("cold should be below min_score, got %+v", h)
		}
	}
}
