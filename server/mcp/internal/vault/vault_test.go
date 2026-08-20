package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMD(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNoteIDKeepsMdAndSlash(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "文章", "foo.md")
	id := NoteID(root, abs)
	if id != "文章/foo.md" {
		t.Fatalf("id = %q", id)
	}
	if strings.Contains(id, `\`) {
		t.Fatal("backslash in id")
	}
}

func TestClassifyKind(t *testing.T) {
	cases := map[string]struct {
		kind string
		skip bool
	}{
		"文章/a.md":                   {"article", false},
		"项目/p.md":                   {"project", false},
		"友链/x.md":                   {"", true},
		"Workbase/context/p.md":     {"context_pack", false},
		"Workbase/skills/s.md":      {"skill", false},
		"Workbase/mcps/m.md":        {"mcp_server", false},
		"Workbase/conventions/x.md": {"note", false},
		"部署溯源/d.md":                 {"note", false},
	}
	for rel, want := range cases {
		got, skip := ClassifyKind(rel)
		if got != want.kind || skip != want.skip {
			t.Fatalf("%s: kind=%q skip=%v want %q/%v", rel, got, skip, want.kind, want.skip)
		}
	}
}

func TestScanKindAndVisibility(t *testing.T) {
	root := t.TempDir()
	writeMD(t, root, "文章/hello.md", "---\nsummary: hi\n---\n# Hello\n\n[[secret-note]]\n")
	writeMD(t, root, "项目/pulse.md", "---\nname: Pulse\nsummary: go agent\nstatus: 维护中\n---\n# Pulse\n")
	writeMD(t, root, "友链/site.md", "---\nname: x\nurl: https://x.test\n---\n")
	writeMD(t, root, "Workbase/skills/lint.md", "---\nid: markdown-lint\nkind: skill\nname: Lint\nsummary: s\n---\n# Lint\n")
	writeMD(t, root, "Workbase/context/profile.md", "---\nid: profile\nkind: context_pack\ntitle: 我\nstartup: true\npriority: high\n---\n# 我\n")
	writeMD(t, root, "Workbase/mcps/wb.md", "---\nid: jiangnan-workbase\nkind: mcp_server\nname: WB\nsummary: s\ntransport: streamable-http\n---\n# WB\n")
	writeMD(t, root, "Workbase/oops.md", "# stray\n")
	writeMD(t, root, "部署溯源/run.md", "---\nvisibility: secret\n---\n# secret-note\n")
	writeMD(t, root, ".obsidian/app.md", "# skip\n")

	idx, err := Scan(root, []string{".obsidian", ".trash"}, map[string]string{
		"文章":               "public",
		"项目":               "public",
		"友链":               "public",
		"部署溯源":             "private",
		"Workbase/context": "private",
		"Workbase/skills":  "private",
		"Workbase/mcps":    "private",
		"default":          "private",
	})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Note{}
	for _, n := range idx.Notes {
		byID[n.ID] = n
	}
	if _, ok := byID["友链/site.md"]; ok {
		t.Fatal("friends should not be indexed")
	}
	if _, ok := byID[".obsidian/app.md"]; ok {
		t.Fatal("obsidian should not be indexed")
	}
	art := byID["文章/hello.md"]
	if art.Kind != "article" || art.Visibility != "public" || !strings.HasSuffix(art.ID, ".md") {
		t.Fatalf("article = %+v", art)
	}
	if byID["项目/pulse.md"].Kind != "project" {
		t.Fatal("project kind")
	}
	if byID["Workbase/skills/lint.md"].Kind != "skill" {
		t.Fatal("skill kind")
	}
	if byID["Workbase/context/profile.md"].Kind != "context_pack" {
		t.Fatal("context kind")
	}
	if byID["Workbase/oops.md"].Kind != "note" || byID["Workbase/oops.md"].Visibility != "private" {
		t.Fatalf("stray = %+v", byID["Workbase/oops.md"])
	}
	if byID["部署溯源/run.md"].Visibility != "secret" {
		t.Fatal("explicit secret")
	}
	if ToolID(byID["项目/pulse.md"]) != "pulse" {
		t.Fatalf("project tool id = %s", ToolID(byID["项目/pulse.md"]))
	}
	if ToolID(byID["Workbase/skills/lint.md"]) != "markdown-lint" {
		t.Fatal("skill tool id")
	}
	if len(idx.Projects) != 1 || len(idx.Skills) != 1 || len(idx.MCPS) != 1 || len(idx.Context) != 1 {
		t.Fatalf("typed slices p=%d s=%d m=%d c=%d", len(idx.Projects), len(idx.Skills), len(idx.MCPS), len(idx.Context))
	}
}

func TestWikiLinkAmbiguousNoEdge(t *testing.T) {
	root := t.TempDir()
	writeMD(t, root, "文章/foo.md", "# A\n[[foo]]\n")
	writeMD(t, root, "部署溯源/foo.md", "# B\n")
	idx, err := Scan(root, nil, map[string]string{"文章": "public", "default": "private"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Links) != 0 {
		t.Fatalf("ambiguous should not create edge, got %+v", idx.Links)
	}
}

func TestWikiLinkPathMatch(t *testing.T) {
	root := t.TempDir()
	writeMD(t, root, "文章/foo.md", "# A\n[[部署溯源/foo]]\n")
	writeMD(t, root, "部署溯源/foo.md", "# B\n")
	idx, err := Scan(root, nil, map[string]string{"文章": "public", "default": "private"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Links) != 1 || idx.Links[0].TargetID != "部署溯源/foo.md" {
		t.Fatalf("links = %+v", idx.Links)
	}
}

func TestMatchScopeAndVisibleInList(t *testing.T) {
	if !VisibleInList("public") || !VisibleInList("private") || !VisibleInList("draft") {
		t.Fatal("list should include public/private/draft")
	}
	if VisibleInList("secret") {
		t.Fatal("list must hide secret")
	}
	if !MatchScope("draft", "all") || MatchScope("secret", "all") {
		t.Fatal("scope all")
	}
	if !MatchScope("public", "public") || MatchScope("draft", "public") {
		t.Fatal("scope public")
	}
	if !MatchScope("private", "private") || MatchScope("draft", "private") {
		t.Fatal("scope private")
	}
}

func TestResolveVisibility(t *testing.T) {
	d := map[string]string{"文章": "public", "Workbase/skills": "private", "default": "private"}
	if got := ResolveVisibility("文章/a.md", "", d); got != "public" {
		t.Fatalf("got %s", got)
	}
	if got := ResolveVisibility("文章/a.md", "draft", d); got != "draft" {
		t.Fatalf("got %s", got)
	}
	if got := ResolveVisibility("文章/a.md", "secret", d); got != "secret" {
		t.Fatalf("got %s", got)
	}
	if got := ResolveVisibility("Workbase/skills/x.md", "", d); got != "private" {
		t.Fatalf("got %s", got)
	}
}
