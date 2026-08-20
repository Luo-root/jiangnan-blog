package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/config"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/inbox"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

func setupWrite(t *testing.T) (*depsHolder, string) {
	t.Helper()
	root := t.TempDir()
	writeVault(t, root, "部署溯源/n.md", "# N\n\nbody\n")
	writeVault(t, root, "文章/hello.md", "# Hello\n")

	prop, err := proposal.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inboxDir := t.TempDir()
	box, err := inbox.New(inboxDir, 7)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(box.Close)

	cfg := &config.Config{}
	cfg.Schema.SensitivePatterns = []string{}
	return &depsHolder{d: Deps{
		Proposal:  prop,
		Inbox:     box,
		VaultRoot: root,
		Cfg:       cfg,
	}}, inboxDir
}

func createArgs(targetType, path, op, content string) map[string]any {
	return map[string]any{
		"target":    map[string]any{"type": targetType, "path": path},
		"operation": map[string]any{"type": op},
		"payload":   map[string]any{"format": "markdown", "content": content},
		"reason":    "test",
	}
}

func TestProposalCreateOK(t *testing.T) {
	r, _ := setupWrite(t)
	m := wrap(r.handleProposalCreate(context.Background(), callReq(createArgs("note", "部署溯源/n.md", "append", "tail")))).mapOK(t)
	id, _ := m["id"].(string)
	if !strings.HasPrefix(id, "prop_") {
		t.Fatalf("id = %v", m["id"])
	}
	if m["status"] != proposal.StatusPending && m["status"] != "pending" {
		t.Fatalf("status = %v", m["status"])
	}
	got, err := r.d.Proposal.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "note" {
		t.Fatalf("kind = %s", got.Kind)
	}
	if got.Target.Path != "部署溯源/n.md" {
		t.Fatalf("path = %s", got.Target.Path)
	}
}

func TestProposalCreateOperationNotSupported(t *testing.T) {
	r, _ := setupWrite(t)
	msg := wrap(r.handleProposalCreate(context.Background(), callReq(createArgs("note", "部署溯源/n.md", "create_file", "# x\n")))).errText(t)
	if !strings.Contains(msg, "operation_not_supported") {
		t.Fatalf("got %q", msg)
	}
	props, _ := r.d.Proposal.List()
	if len(props) != 0 {
		t.Fatalf("should not create receipt, got %d", len(props))
	}
}

func TestProposalCreateStaleBase(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	r, _ := setupWrite(t)
	gitDir := initVaultGit(t, r.d.VaultRoot)
	r.d.GitDir = gitDir

	args := createArgs("note", "部署溯源/n.md", "append", "tail")
	args["expected_base"] = "deadbeef"
	msg := wrap(r.handleProposalCreate(context.Background(), callReq(args))).errText(t)
	if !strings.Contains(msg, "stale_base") || !strings.Contains(msg, "你读的已经不是最新") {
		t.Fatalf("got %q", msg)
	}
	props, _ := r.d.Proposal.List()
	if len(props) != 0 {
		t.Fatalf("stale_base must not write receipt, got %d", len(props))
	}
}

func TestProposalCreateSensitiveWarning(t *testing.T) {
	r, _ := setupWrite(t)
	r.d.Cfg.Schema.SensitivePatterns = []string{`-----BEGIN [A-Z ]*PRIVATE KEY-----`}
	m := wrap(r.handleProposalCreate(context.Background(), callReq(createArgs(
		"note", "部署溯源/n.md", "append", "-----BEGIN RSA PRIVATE KEY-----\nxxx\n",
	)))).mapOK(t)
	val, ok := m["validation"].(proposal.Validation)
	if !ok {
		t.Fatalf("validation = %T %#v", m["validation"], m["validation"])
	}
	if !val.OK {
		t.Fatal("sensitive hit must still create")
	}
	if len(val.Warnings) == 0 {
		t.Fatal("want warnings")
	}
	if m["status"] != proposal.StatusPending && m["status"] != "pending" {
		t.Fatalf("status = %v", m["status"])
	}
}

func TestInboxAppendWarningNotPersisted(t *testing.T) {
	r, inboxDir := setupWrite(t)
	r.d.Cfg.Schema.SensitivePatterns = []string{`-----BEGIN [A-Z ]*PRIVATE KEY-----`}
	content := "-----BEGIN RSA PRIVATE KEY-----\nxxx\n"
	m := wrap(r.handleInboxAppend(context.Background(), callReq(map[string]any{
		"content": content,
		"title":   "secret-todo",
	}))).mapOK(t)
	ws, _ := m["warnings"].([]string)
	if len(ws) == 0 {
		t.Fatalf("want response warnings, got %#v", m["warnings"])
	}
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	b, err := os.ReadFile(filepath.Join(inboxDir, id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "warnings") {
		t.Fatalf("warnings must not be written to md:\n%s", b)
	}
	if !strings.Contains(string(b), content) {
		t.Fatalf("content missing from md:\n%s", b)
	}

	got := wrap(r.handleInboxGet(context.Background(), callReq(map[string]any{"id": id}))).mapOK(t)
	gws, _ := got["warnings"].([]string)
	if len(gws) == 0 {
		t.Fatal("inbox.get should rescan warnings")
	}
}

func initVaultGit(t *testing.T, work string) string {
	t.Helper()
	gitDir := filepath.Join(t.TempDir(), "git.git")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"--git-dir=" + gitDir, "--work-tree=" + work}, args...)...)
		cmd.Dir = work
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "core.autocrlf", "false")
	run("config", "user.name", "t")
	run("config", "user.email", "t@t")
	run("add", ".")
	run("commit", "-m", "init")
	return gitDir
}
