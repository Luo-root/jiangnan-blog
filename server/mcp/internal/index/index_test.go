package index

import (
	"os"
	"path/filepath"
	"testing"
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
