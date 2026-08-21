package admin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/audit"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/index"
)

func loginToken(t *testing.T, h *Handler) string {
	t.Helper()
	w := doJSON(h, http.MethodPost, "/api/admin/login", loginReq{User: "jiangnan", Password: "secret"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var sess loginResp
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	return sess.Token
}

func writeVaultFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func pagesHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	root := t.TempDir()
	writeVaultFile(t, root, "文章/hello.md", "---\nsummary: kubernetes article\n---\n# Hello kubernetes\n")
	writeVaultFile(t, root, "项目/pulse.md", "---\nname: Pulse\nsummary: kubernetes project\n---\n# Pulse\n")
	writeVaultFile(t, root, "Workbase/skills/lint.md", "---\nid: markdown-lint\nkind: skill\nname: Lint\nsummary: kubernetes skill\n---\n# Lint\n")
	writeVaultFile(t, root, "部署溯源/private-note.md", "---\nsummary: private kubernetes note\n---\n# Private kubernetes\n")

	idx, err := index.Open(filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	if err := idx.Rebuild(root, []string{".obsidian", ".trash"}, map[string]string{
		"文章": "public", "项目": "public", "部署溯源": "private", "Workbase/skills": "private", "default": "private",
	}); err != nil {
		t.Fatal(err)
	}

	aud, err := audit.Open(filepath.Join(t.TempDir(), "audit.sqlite"), 90, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = aud.Close() })
	aud.Append(audit.Entry{Tool: "knowledge.search", ClientID: "coder", ResultStatus: "success", DurationMS: 12, ArgsDigest: "abc"})

	h := testHandler()
	h.Index = idx
	h.Audit = aud
	h.VaultRoot = root
	h.RuntimeDir = t.TempDir()
	h.TemplatesDir = filepath.Join(h.RuntimeDir, "templates")
	h.MCPListen = "127.0.0.1:8787"
	h.AdminListen = "127.0.0.1:8788"
	return h, loginToken(t, h)
}

func TestNewAdminAPIsRequireSession(t *testing.T) {
	h := testHandler()
	for _, path := range []string{
		"/api/audit/recent",
		"/api/knowledge/search?q=k",
		"/api/knowledge?id=x",
		"/api/system/health",
		"/api/git/history",
		"/api/templates",
		"/api/agent-prompt",
	} {
		w := doJSON(h, http.MethodGet, path, nil, "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
}

func TestAuditRecentAndNilStore(t *testing.T) {
	h, token := pagesHandler(t)
	w := doJSON(h, http.MethodGet, "/api/audit/recent?tool=knowledge.search&client_id=coder", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("audit: %d %s", w.Code, w.Body.String())
	}
	var items []audit.Entry
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Tool != "knowledge.search" || items[0].ClientID != "coder" {
		t.Fatalf("items=%+v", items)
	}

	empty := testHandler()
	tok := loginToken(t, empty)
	w = doJSON(empty, http.MethodGet, "/api/audit/recent", nil, tok)
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("nil store: %d %s", w.Code, w.Body.String())
	}
}

func TestKnowledgeSearchCrossKindAndGet(t *testing.T) {
	h, token := pagesHandler(t)
	w := doJSON(h, http.MethodGet, "/api/knowledge/search?q=kubernetes", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("search: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Results []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, r := range out.Results {
		ids[r.ID] = r.Kind
	}
	if ids["文章/hello.md"] != "article" || ids["项目/pulse.md"] != "project" || ids["Workbase/skills/lint.md"] != "skill" {
		t.Fatalf("admin search must cross kind, got %v", ids)
	}

	w = doJSON(h, http.MethodGet, "/api/knowledge/search?q=kubernetes&kind=project", nil, token)
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 || out.Results[0].ID != "项目/pulse.md" {
		t.Fatalf("kind=project: %+v", out.Results)
	}

	w = doJSON(h, http.MethodGet, "/api/knowledge/search?q=zzzz-not-found", nil, token)
	var empty map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if empty["message"] != "未查询到相关内容" {
		t.Fatalf("empty = %v", empty)
	}

	w = doJSON(h, http.MethodGet, "/api/knowledge?id="+url.QueryEscape("文章/hello.md"), nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	var note map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &note); err != nil {
		t.Fatal(err)
	}
	if note["id"] != "文章/hello.md" || note["kind"] != "article" {
		t.Fatalf("note=%v", note)
	}
}

func TestSystemHealth(t *testing.T) {
	h, token := pagesHandler(t)
	w := doJSON(h, http.MethodGet, "/api/system/health", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != true {
		t.Fatalf("ok=%v", got["ok"])
	}
	idx, _ := got["index"].(map[string]any)
	if idx["notes"] == nil {
		t.Fatalf("index=%v", got["index"])
	}
}

func TestTemplatesCRUD(t *testing.T) {
	h, token := pagesHandler(t)
	w := doJSON(h, http.MethodPost, "/api/templates", map[string]any{
		"name": "append note", "reason": "r", "target_type": "note", "operation": "append", "payload": "x",
		"scopes": []string{"write:proposal"},
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created Template
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Name != "append note" {
		t.Fatalf("created=%+v", created)
	}
	if created.Kind != "proposal" {
		t.Fatalf("default kind=%s", created.Kind)
	}

	w = doJSON(h, http.MethodPost, "/api/templates", map[string]any{
		"name": "inbox todo", "kind": "inbox", "title": "t", "content": "c",
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("inbox tpl: %d %s", w.Code, w.Body.String())
	}
	var inboxTpl Template
	if err := json.Unmarshal(w.Body.Bytes(), &inboxTpl); err != nil {
		t.Fatal(err)
	}
	if inboxTpl.Kind != "inbox" {
		t.Fatalf("kind=%s", inboxTpl.Kind)
	}

	w = doJSON(h, http.MethodGet, "/api/templates", nil, token)
	var list []Template
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list=%+v", list)
	}

	w = doJSON(h, http.MethodPost, "/api/templates/"+created.ID, map[string]any{"payload": "y", "reason": "updated"}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	var updated Template
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Payload != "y" || updated.Reason != "updated" {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestGitHistoryAndDiff(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init")
	run("config", "user.name", "t")
	run("config", "user.email", "t@t")
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.md")
	run("-c", "user.name=t", "-c", "user.email=t@t", "commit", "-m", "first")
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("two\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.md")
	run("-c", "user.name=t", "-c", "user.email=t@t", "commit", "-m", "second")

	h := testHandler()
	h.GitDir = filepath.Join(dir, ".git")
	token := loginToken(t, h)
	w := doJSON(h, http.MethodGet, "/api/git/history?limit=10", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("history: %d %s", w.Code, w.Body.String())
	}
	var commits []struct {
		SHA     string `json:"sha"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &commits); err != nil {
		t.Fatal(err)
	}
	if len(commits) < 2 || commits[0].Subject != "second" {
		t.Fatalf("commits=%+v", commits)
	}
	w = doJSON(h, http.MethodGet, "/api/git/diff/"+commits[0].SHA, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("diff: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "two") {
		t.Fatalf("diff body=%s", w.Body.String())
	}

	w = doJSON(h, http.MethodGet, "/api/git/diff/--output=/tmp/x", nil, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("flag as sha: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(h, http.MethodGet, "/api/git/diff/HEAD", nil, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HEAD as sha: %d %s", w.Code, w.Body.String())
	}
}

func TestSearchEmptyQueryNeedsFilter(t *testing.T) {
	h, token := pagesHandler(t)
	w := doJSON(h, http.MethodGet, "/api/knowledge/search", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("empty: %d %s", w.Code, w.Body.String())
	}
	var empty map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if empty["message"] != "输入关键词或选一个过滤条件" {
		t.Fatalf("empty = %v", empty)
	}

	w = doJSON(h, http.MethodGet, "/api/knowledge/search?kind=article", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("kind only: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Results []struct {
			ID      string             `json:"id"`
			Score   float64            `json:"score"`
			Signals map[string]float64 `json:"signals"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) == 0 {
		t.Fatal("kind=article should list")
	}
	for _, r := range out.Results {
		if r.Score != 0 {
			t.Fatalf("filter-list score should be 0, got %+v", r)
		}
		if r.Signals == nil {
			t.Fatalf("filter-list should still carry access/recency signals: %+v", r)
		}
	}

	w = doJSON(h, http.MethodGet, "/api/knowledge/search?kind=article&sort=access", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("sort=access: %d %s", w.Code, w.Body.String())
	}
}

func TestAuditUntilRange(t *testing.T) {
	h, token := pagesHandler(t)
	old := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	h.Audit.Append(audit.Entry{TS: old, Tool: "old", ClientID: "coder", ResultStatus: "success"})
	h.Audit.Append(audit.Entry{TS: mid, Tool: "mid", ClientID: "coder", ResultStatus: "success"})
	h.Audit.Append(audit.Entry{TS: now, Tool: "now", ClientID: "coder", ResultStatus: "success"})

	w := doJSON(h, http.MethodGet, "/api/audit/recent?since=2026-08-10T09:00:00Z&until=2026-08-10T11:00:00Z", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("range: %d %s", w.Code, w.Body.String())
	}
	var items []audit.Entry
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Tool != "mid" {
		t.Fatalf("items=%+v", items)
	}

	w = doJSON(h, http.MethodGet, "/api/audit/recent?since=2026-08-20T00:00:00Z&until=2026-08-01T00:00:00Z", nil, token)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_argument") {
		t.Fatalf("until < since: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(h, http.MethodGet, "/api/audit/recent?until=not-a-date", nil, token)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_argument: until") {
		t.Fatalf("bad until: %d %s", w.Code, w.Body.String())
	}
}

func TestAgentPromptCopy(t *testing.T) {
	h, token := pagesHandler(t)
	writeVaultFile(t, h.VaultRoot, "Workbase/mcps/jiangnan-workbase.md", `---
id: jiangnan-workbase
kind: mcp_server
name: Jiangnan Workbase MCP
summary: 私密个人 Agent 工作基座。
visibility: private
---

# Jiangnan Workbase MCP

## Purpose

以 Obsidian Vault 为事实源。

## Security

- 挡未授权的是 token + scope + visibility

## Source

https://github.com/Luo-root/jiangnan-blog-agent-workbase
`)
	h.WorkbaseRoot = filepath.Join(h.VaultRoot, "Workbase")
	w := doJSON(h, http.MethodGet, "/api/agent-prompt", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("prompt: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Prompt, "# Jiangnan Workbase MCP") || !strings.Contains(out.Prompt, "workbase.identity") {
		t.Fatalf("prompt = %s", out.Prompt)
	}
}
