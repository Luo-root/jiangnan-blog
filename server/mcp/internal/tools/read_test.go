package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/index"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/search"
)

func writeVault(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupRead(t *testing.T) *depsHolder {
	t.Helper()
	root := t.TempDir()
	writeVault(t, root, "文章/hello.md", "---\nsummary: kubernetes article\n---\n# Hello kubernetes\n")
	writeVault(t, root, "文章/draft.md", "---\nvisibility: draft\nsummary: draft kubernetes\n---\n# Draft kubernetes\n")
	writeVault(t, root, "部署溯源/secret.md", "---\nvisibility: secret\nsummary: secret kubernetes\n---\n# Secret kubernetes\n")
	writeVault(t, root, "部署溯源/private-note.md", "---\nsummary: private kubernetes note\n---\n# Private kubernetes\n")
	writeVault(t, root, "项目/pulse.md", "---\nname: Pulse\nsummary: kubernetes project\n---\n# Pulse\n")
	writeVault(t, root, "项目/hidden.md", "---\nname: Hidden\nvisibility: secret\nsummary: secret project kubernetes\n---\n# Hidden\n")
	writeVault(t, root, "Workbase/skills/lint.md", "---\nid: markdown-lint\nkind: skill\nname: Lint\nsummary: kubernetes skill\n---\n# Lint\n")
	writeVault(t, root, "Workbase/skills/secret-skill.md", "---\nid: secret-skill\nkind: skill\nname: SecretSkill\nvisibility: secret\nsummary: kubernetes secret skill\n---\n# SecretSkill\n")
	writeVault(t, root, "Workbase/context/profile.md", "---\nid: profile\nkind: context_pack\ntitle: 我\nstartup: true\npriority: high\n---\n# 我\nstartup pack\n")
	writeVault(t, root, "Workbase/context/secret.md", "---\nid: secret-pack\nkind: context_pack\ntitle: Secret\nstartup: true\npriority: high\nvisibility: secret\n---\n# Secret pack\n")
	writeVault(t, root, "Workbase/mcps/wb.md", "---\nid: jiangnan-workbase\nkind: mcp_server\nname: WB\nsummary: s\ntransport: streamable-http\n---\n# WB\n")
	writeVault(t, root, "Workbase/oops.md", "---\nsummary: stray kubernetes note\n---\n# stray\n")

	idx, err := index.Open(filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	if err := idx.Rebuild(root, []string{".obsidian", ".trash"}, map[string]string{
		"文章":               "public",
		"项目":               "public",
		"友链":               "public",
		"部署溯源":             "private",
		"Workbase/context": "private",
		"Workbase/skills":  "private",
		"Workbase/mcps":    "private",
		"default":          "private",
	}); err != nil {
		t.Fatal(err)
	}
	return &depsHolder{d: Deps{Idx: idx, VaultRoot: root}}
}

func callReq(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

type toolOut struct {
	res *mcp.CallToolResult
	err error
}

func wrap(res *mcp.CallToolResult, err error) toolOut {
	return toolOut{res: res, err: err}
}

func errorText(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			return v.Text
		case *mcp.TextContent:
			return v.Text
		}
	}
	return ""
}

func (o toolOut) mapOK(t *testing.T) map[string]any {
	t.Helper()
	if o.err != nil {
		t.Fatal(o.err)
	}
	if o.res.IsError {
		t.Fatalf("unexpected error: %s", errorText(o.res))
	}
	m, ok := o.res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured = %T %#v", o.res.StructuredContent, o.res.StructuredContent)
	}
	return m
}

func (o toolOut) errText(t *testing.T) string {
	t.Helper()
	if o.err != nil {
		t.Fatal(o.err)
	}
	if !o.res.IsError {
		t.Fatalf("want error, got %+v", o.res.StructuredContent)
	}
	return errorText(o.res)
}

func searchIDs(m map[string]any) []string {
	switch rs := m["results"].(type) {
	case []search.Result:
		out := make([]string, 0, len(rs))
		for _, h := range rs {
			out = append(out, h.ID)
		}
		return out
	case []any:
		out := make([]string, 0, len(rs))
		for _, item := range rs {
			switch h := item.(type) {
			case search.Result:
				out = append(out, h.ID)
			case map[string]any:
				if id, ok := h["id"].(string); ok {
					out = append(out, id)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func hasID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func asMaps(v any) []map[string]any {
	switch items := v.(type) {
	case []map[string]any:
		return items
	case []map[string]string:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			m := make(map[string]any, len(item))
			for k, val := range item {
				m[k] = val
			}
			out = append(out, m)
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func TestParseSearchKinds(t *testing.T) {
	kinds, given := parseSearchKinds(nil)
	if given || len(kinds) != 2 {
		t.Fatalf("nil args: %v given=%v", kinds, given)
	}
	kinds, given = parseSearchKinds(map[string]any{})
	if given {
		t.Fatal("missing kind should not be given")
	}
	kinds, given = parseSearchKinds(map[string]any{"kind": []any{}})
	if !given || len(kinds) != 0 {
		t.Fatalf("empty kind: %v given=%v", kinds, given)
	}
	kinds, given = parseSearchKinds(map[string]any{"kind": []any{"project"}})
	if !given || len(kinds) != 0 {
		t.Fatalf("project only: %v given=%v", kinds, given)
	}
	kinds, given = parseSearchKinds(map[string]any{"kind": []any{"article", "project"}})
	if !given || len(kinds) != 1 || kinds[0] != "article" {
		t.Fatalf("mixed: %v", kinds)
	}
}

func TestKnowledgeSearchDefaultKindAndSecret(t *testing.T) {
	r := setupRead(t)
	m := wrap(r.handleKnowledgeSearch(context.Background(), callReq(map[string]any{"query": "kubernetes"}))).mapOK(t)
	ids := searchIDs(m)
	if !hasID(ids, "文章/hello.md") || !hasID(ids, "文章/draft.md") || !hasID(ids, "部署溯源/private-note.md") || !hasID(ids, "Workbase/oops.md") {
		t.Fatalf("default search ids = %v", ids)
	}
	if hasID(ids, "部署溯源/secret.md") {
		t.Fatal("secret leaked into search")
	}
	if hasID(ids, "项目/pulse.md") || hasID(ids, "Workbase/skills/lint.md") {
		t.Fatalf("project/skill leaked: %v", ids)
	}
}

func TestKnowledgeSearchKindEmptySet(t *testing.T) {
	r := setupRead(t)
	ctx := context.Background()
	for _, kind := range []any{[]any{}, []any{"project"}, []string{"mcp_server"}} {
		m := wrap(r.handleKnowledgeSearch(ctx, callReq(map[string]any{"query": "kubernetes", "kind": kind}))).mapOK(t)
		if len(searchIDs(m)) != 0 {
			t.Fatalf("kind=%v results=%v", kind, m["results"])
		}
		if m["message"] != "未查询到相关内容" {
			t.Fatalf("message = %v", m["message"])
		}
		if m["query_echo"] != "kubernetes" {
			t.Fatalf("query_echo = %v", m["query_echo"])
		}
	}
}

func TestKnowledgeSearchKindArticleDropsProject(t *testing.T) {
	r := setupRead(t)
	m := wrap(r.handleKnowledgeSearch(context.Background(), callReq(map[string]any{
		"query": "kubernetes",
		"kind":  []any{"article", "project"},
	}))).mapOK(t)
	ids := searchIDs(m)
	if !hasID(ids, "文章/hello.md") || hasID(ids, "项目/pulse.md") {
		t.Fatalf("ids = %v", ids)
	}
}

func TestKnowledgeSearchInvalidIntentAndScope(t *testing.T) {
	r := setupRead(t)
	ctx := context.Background()
	got := wrap(r.handleKnowledgeSearch(ctx, callReq(map[string]any{"query": "k", "intent": "whatever"}))).errText(t)
	if !strings.Contains(got, "invalid_argument: intent") {
		t.Fatalf("intent err = %q", got)
	}
	got = wrap(r.handleKnowledgeSearch(ctx, callReq(map[string]any{"query": "k", "scope": "secret"}))).errText(t)
	if !strings.Contains(got, "invalid_argument: scope") {
		t.Fatalf("scope err = %q", got)
	}
}

func TestKnowledgeSearchScope(t *testing.T) {
	r := setupRead(t)
	ctx := context.Background()
	pub := searchIDs(wrap(r.handleKnowledgeSearch(ctx, callReq(map[string]any{"query": "kubernetes", "scope": "public"}))).mapOK(t))
	if !hasID(pub, "文章/hello.md") || hasID(pub, "文章/draft.md") || hasID(pub, "部署溯源/private-note.md") {
		t.Fatalf("public = %v", pub)
	}
	priv := searchIDs(wrap(r.handleKnowledgeSearch(ctx, callReq(map[string]any{"query": "kubernetes", "scope": "private"}))).mapOK(t))
	if !hasID(priv, "部署溯源/private-note.md") || hasID(priv, "文章/hello.md") || hasID(priv, "文章/draft.md") {
		t.Fatalf("private = %v", priv)
	}
}

func TestKnowledgeGetIDAndSecret(t *testing.T) {
	r := setupRead(t)
	ctx := context.Background()

	ok := wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": "文章/hello.md"}))).mapOK(t)
	if ok["id"] != "文章/hello.md" || ok["kind"] != "article" {
		t.Fatalf("get = %+v", ok)
	}

	native := wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": filepath.FromSlash("文章/hello.md")}))).mapOK(t)
	if native["id"] != "文章/hello.md" {
		t.Fatalf("native id = %v", native["id"])
	}

	draft := wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": "文章/draft.md"}))).mapOK(t)
	if draft["visibility"] != "draft" {
		t.Fatalf("draft = %+v", draft)
	}

	got := wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": "文章/hello"}))).errText(t)
	if !strings.Contains(got, "note not found") {
		t.Fatalf("missing .md = %q", got)
	}
	got = wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": "markdown-lint"}))).errText(t)
	if !strings.Contains(got, "note not found") {
		t.Fatalf("skill id = %q", got)
	}
	got = wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": "Workbase/skills/lint.md"}))).errText(t)
	if !strings.Contains(got, "note not found") {
		t.Fatalf("skill path = %q", got)
	}
	got = wrap(r.handleKnowledgeGet(ctx, callReq(map[string]any{"id": "部署溯源/secret.md"}))).errText(t)
	if got != "secret_blocked" {
		t.Fatalf("secret = %q", got)
	}
}

func TestListAndStartupHideSecret(t *testing.T) {
	r := setupRead(t)
	ctx := context.Background()

	startup := wrap(r.handleContextStartup(ctx, callReq(nil))).mapOK(t)
	packs := asMaps(startup["packs"])
	if len(packs) != 1 || packs[0]["id"] != "profile" {
		t.Fatalf("startup packs = %+v", startup["packs"])
	}

	projects := wrap(r.handleProjectList(ctx, callReq(nil))).mapOK(t)
	for _, item := range asMaps(projects["projects"]) {
		if item["id"] == "hidden" || item["visibility"] == "secret" {
			t.Fatalf("secret project listed: %+v", item)
		}
	}

	skills := wrap(r.handleSkillList(ctx, callReq(nil))).mapOK(t)
	for _, item := range asMaps(skills["skills"]) {
		if item["id"] == "secret-skill" {
			t.Fatalf("secret skill listed: %+v", item)
		}
	}

	got := wrap(r.handleSkillGet(ctx, callReq(map[string]any{"id": "secret-skill"}))).errText(t)
	if got != "secret_blocked" {
		t.Fatalf("skill secret = %q", got)
	}
	got = wrap(r.handleContextGet(ctx, callReq(map[string]any{"id": "secret-pack"}))).errText(t)
	if got != "secret_blocked" {
		t.Fatalf("context secret = %q", got)
	}
}
