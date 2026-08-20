package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

func TestAllowedMatrix(t *testing.T) {
	if !Allowed("note", "append") || !Allowed("note", "append_section") {
		t.Fatal("note ops")
	}
	if Allowed("note", "create_file") || Allowed("article", "append") {
		t.Fatal("matrix leak")
	}
	if !Allowed("article", "create_file") || !Allowed("skill", "register_item") {
		t.Fatal("create/register")
	}
	if Allowed("note", "replace_frontmatter") {
		t.Fatal("replace_frontmatter is not in the matrix")
	}
}

func TestResolvePath(t *testing.T) {
	root := t.TempDir()
	abs, slash, err := ResolvePath(root, `部署溯源\foo.md`)
	if err != nil {
		t.Fatal(err)
	}
	if slash != "部署溯源/foo.md" {
		t.Fatalf("slash = %s", slash)
	}
	if !strings.HasPrefix(abs, root) {
		t.Fatalf("abs %s not under root", abs)
	}
	if _, _, err := ResolvePath(root, "../etc/passwd"); err == nil {
		t.Fatal("expected target_path_invalid")
	}
}

func TestApplyOpAppendAndSection(t *testing.T) {
	base := "# Title\n\n## Agent Workbase MCP\n\nold\n\n## Other\n\nx\n"
	got, err := ApplyOp(base, proposal.Operation{Type: "append", Section: ""}, "tail")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "tail") {
		t.Fatalf("append = %q", got)
	}
	got, err = ApplyOp(base, proposal.Operation{Type: "append_section", Section: "Agent Workbase MCP"}, "v0.1 完成。")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "v0.1 完成。") || !strings.Contains(got, "## Other") {
		t.Fatalf("append_section = %q", got)
	}
	got, err = ApplyOp(base, proposal.Operation{Type: "append_section", Section: "New Section"}, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "## New Section") || !strings.Contains(got, "hello") {
		t.Fatalf("create heading = %q", got)
	}
	got, err = ApplyOp(base, proposal.Operation{Type: "patch_section", Section: "Agent Workbase MCP"}, "replaced")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "replaced") || strings.Contains(got, "old") {
		t.Fatalf("patch = %q", got)
	}
	if _, err := ApplyOp(base, proposal.Operation{Type: "patch_section", Section: "Missing"}, "x"); err != ErrSectionNotFound {
		t.Fatalf("want section_not_found, got %v", err)
	}
}

func TestApplyCreateAndAppend(t *testing.T) {
	root := t.TempDir()
	p := &proposal.Proposal{
		ID: "prop_20260820_001",
		Target: proposal.Target{
			Type: "article",
			Path: "文章/new.md",
		},
		Operation: proposal.Operation{Type: "create_file"},
		Payload:   proposal.Payload{Format: "markdown", Content: "# Hello\n"},
	}
	r, err := Apply(p, Deps{VaultRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != proposal.StatusApplied || r.MergeStrategy != "none" {
		t.Fatalf("receipt = %+v", r)
	}
	b, _ := os.ReadFile(filepath.Join(root, "文章", "new.md"))
	if !strings.Contains(string(b), "# Hello") {
		t.Fatalf("file = %q", b)
	}

	_, err = Apply(p, Deps{VaultRoot: root})
	if err == nil || !strings.Contains(err.Error(), "target_already_exists") {
		t.Fatalf("want exists, got %v", err)
	}

	note := filepath.Join(root, "部署溯源", "n.md")
	if err := os.MkdirAll(filepath.Dir(note), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("# N\n\nbody\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ap := &proposal.Proposal{
		ID: "prop_20260820_002",
		Target: proposal.Target{
			Type: "note",
			Path: "部署溯源/n.md",
		},
		Operation: proposal.Operation{Type: "append"},
		Payload:   proposal.Payload{Content: "more"},
	}
	r, err = Apply(ap, Deps{VaultRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != proposal.StatusApplied {
		t.Fatalf("append receipt = %+v", r)
	}
	b, _ = os.ReadFile(note)
	if !strings.Contains(string(b), "more") {
		t.Fatalf("after append = %q", b)
	}

	ap.Receipt = r
	r2, err := Apply(ap, Deps{VaultRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Replayed {
		t.Fatal("want replayed")
	}
	b, _ = os.ReadFile(note)
	if strings.Count(string(b), "more") != 1 {
		t.Fatalf("replay mutated file: %q", b)
	}
}

func gitRepo(t *testing.T) (work, gitDir string, run func(...string)) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	root := t.TempDir()
	gitDir = filepath.Join(root, ".git")
	work = filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0755); err != nil {
		t.Fatal(err)
	}
	run = func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"--git-dir=" + gitDir, "--work-tree=" + work}, args...)...)
		cmd.Dir = work
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "core.autocrlf", "false")
	run("config", "user.name", "t")
	run("config", "user.email", "t@t")
	return work, gitDir, run
}

func TestThreeWayNoConflict(t *testing.T) {
	work, gitDir, run := gitRepo(t)
	note := filepath.Join(work, "部署溯源", "n.md")
	if err := os.MkdirAll(filepath.Dir(note), 0755); err != nil {
		t.Fatal(err)
	}
	// HEAD 改标题，ours 在文末追加：两段不重叠，应能 3-way 干净合并。
	if err := os.WriteFile(note, []byte("# Title\n\nkeep-me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "base")
	base := Head(gitDir)

	if err := os.WriteFile(note, []byte("# Title-HEAD\n\nkeep-me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "head")

	p := &proposal.Proposal{
		ID:         "prop_20260820_010",
		BaseCommit: base,
		Target:     proposal.Target{Type: "note", Path: "部署溯源/n.md"},
		Operation:  proposal.Operation{Type: "append"},
		Payload:    proposal.Payload{Content: "line-ours"},
	}
	r, err := Apply(p, Deps{VaultRoot: work, GitDir: gitDir})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != proposal.StatusApplied || r.MergeStrategy != "three_way" {
		t.Fatalf("receipt = %+v", r)
	}
	b, _ := os.ReadFile(note)
	text := string(b)
	if !strings.Contains(text, "Title-HEAD") || !strings.Contains(text, "line-ours") || !strings.Contains(text, "keep-me") {
		t.Fatalf("merged = %q", text)
	}
}

func TestThreeWayConflict(t *testing.T) {
	work, gitDir, run := gitRepo(t)
	note := filepath.Join(work, "部署溯源", "n.md")
	if err := os.MkdirAll(filepath.Dir(note), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("# N\n\nshared-line\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "base")
	base := Head(gitDir)

	if err := os.WriteFile(note, []byte("# N\n\nCHANGED-HEAD\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "diverge")

	p := &proposal.Proposal{
		ID:         "prop_20260820_011",
		BaseCommit: base,
		Target:     proposal.Target{Type: "note", Path: "部署溯源/n.md"},
		Operation:  proposal.Operation{Type: "append"},
		Payload:    proposal.Payload{Content: "CHANGED-OURS"},
	}
	r, err := Apply(p, Deps{VaultRoot: work, GitDir: gitDir})
	if err == nil && r.Status != proposal.StatusConflict {
		t.Fatalf("want conflict, got %+v", r)
	}
	if r == nil || r.Status != proposal.StatusConflict {
		t.Fatalf("want conflict receipt, got %+v err=%v", r, err)
	}
	if r.MergeStrategy != "three_way" {
		t.Fatalf("strategy = %s", r.MergeStrategy)
	}
	after, _ := os.ReadFile(note)
	if strings.Contains(string(after), "<<<<<<<") {
		t.Fatalf("conflict leaked into worktree: %s", after)
	}
	if !strings.Contains(string(after), "CHANGED-HEAD") {
		t.Fatalf("worktree should stay at HEAD: %s", after)
	}
}

func TestFrontmatterConflict(t *testing.T) {
	if !frontmatterConflict("---\n<<<<<<< HEAD\nx: 1\n---\nbody\n") {
		t.Fatal("want frontmatter conflict")
	}
	if frontmatterConflict("---\nx: 1\n---\n<<<<<<< HEAD\nbody\n") {
		t.Fatal("body conflict is not frontmatter conflict")
	}
	if frontmatterConflict("# no fm\n<<<<<<< HEAD\n") {
		t.Fatal("no frontmatter")
	}
}

func TestFenceClosed(t *testing.T) {
	if !FenceClosed("hi") || !FenceClosed("```go\nx\n```") {
		t.Fatal("closed fences")
	}
	if FenceClosed("```go\nx") {
		t.Fatal("unclosed")
	}
}
