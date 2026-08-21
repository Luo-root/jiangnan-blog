package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/auth"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/config"
)

const sampleMD = `---
id: jiangnan-workbase
kind: mcp_server
name: Jiangnan Workbase MCP
summary: 私密个人 Agent 工作基座。
visibility: private
---

# Jiangnan Workbase MCP

## Purpose

以 Obsidian Vault 为事实源。

跨设备提供一致上下文。

## Security

- 挡未授权的是 token + scope + visibility
- 敏感模式默认关

## Source

https://github.com/Luo-root/jiangnan-blog-agent-workbase
`

func TestParseWorkbaseMD(t *testing.T) {
	meta, err := parseWorkbaseMD(sampleMD)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "Jiangnan Workbase MCP" {
		t.Fatalf("name = %q", meta.Name)
	}
	if !strings.Contains(meta.Description, "工作基座") {
		t.Fatalf("desc = %q", meta.Description)
	}
	if !strings.Contains(meta.GettingStarted, "Obsidian Vault") {
		t.Fatalf("purpose = %q", meta.GettingStarted)
	}
	if len(meta.CriticalRules) != 2 {
		t.Fatalf("rules = %v", meta.CriticalRules)
	}
	if len(meta.SeeAlso) != 1 || !strings.Contains(meta.SeeAlso[0], "jiangnan-blog-agent-workbase") {
		t.Fatalf("see_also = %v", meta.SeeAlso)
	}
}

func TestParseWorkbaseMDMissingPurpose(t *testing.T) {
	_, err := parseWorkbaseMD(strings.Replace(sampleMD, "## Purpose", "## Other", 1))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToolNamesIncludesIdentity(t *testing.T) {
	names := ToolNames()
	found := false
	for _, n := range names {
		if n == "workbase.identity" {
			found = true
		}
		if n == "workbase.manifest" {
			t.Fatal("old workbase.manifest still registered")
		}
	}
	if !found {
		t.Fatal("workbase.identity missing")
	}
	if RequiredScope("workbase.identity") != "" {
		t.Fatal("identity must not require extra scope")
	}
	if RequiredScope("proposal.list") != ScopeWriteProposal {
		t.Fatalf("proposal.list scope = %q", RequiredScope("proposal.list"))
	}
}

func TestAllowedToolsSubset(t *testing.T) {
	got := AllowedTools([]string{"read:context", "read:knowledge"})
	wantHas := map[string]bool{"workbase.identity": true, "context.startup": true, "knowledge.get": true}
	wantNot := map[string]bool{"proposal.create": true, "inbox.append": true}
	set := map[string]bool{}
	for _, n := range got {
		set[n] = true
	}
	for n := range wantHas {
		if !set[n] {
			t.Fatalf("missing %s in %v", n, got)
		}
	}
	for n := range wantNot {
		if set[n] {
			t.Fatalf("unexpected %s in %v", n, got)
		}
	}
}

func TestIdentityReadsVaultAndPolicyLive(t *testing.T) {
	root := t.TempDir()
	writeVault(t, root, "Workbase/mcps/jiangnan-workbase.md", sampleMD)
	cfg := &config.Config{}
	cfg.Schema.VisibilityPolicy = map[string]string{
		"public":  "可公开展示与索引",
		"private": "授权 Agent 可读",
		"secret":  "默认不暴露给远程 MCP",
		"draft":   "草稿",
	}
	r := &depsHolder{d: Deps{
		WorkbaseRoot: filepath.Join(root, "Workbase"),
		Cfg:          cfg,
	}}
	ctx := auth.WithAuth(context.Background(), &auth.AuthContext{
		ClientID:  "bot",
		Scopes:    []string{"read:context"},
		Status:    auth.StatusActive,
		CreatedAt: time.Now(),
	})
	m := wrap(r.handleIdentity(ctx, callReq(nil))).mapOK(t)
	wb, _ := m["workbase"].(map[string]any)
	if wb["name"] != "Jiangnan Workbase MCP" {
		t.Fatalf("name = %v", wb["name"])
	}
	if !strings.Contains(fmtString(wb["description"]), "工作基座") {
		t.Fatalf("desc = %v", wb["description"])
	}
	if got := policyPublic(wb["visibility_policy"]); got != "可公开展示与索引" {
		t.Fatalf("policy = %v", wb["visibility_policy"])
	}

	updated := strings.Replace(sampleMD, "name: Jiangnan Workbase MCP", "name: 即时读取 Workbase", 1)
	updated = strings.Replace(updated, "summary: 私密个人 Agent 工作基座。", "summary: 改完立刻可见。", 1)
	if err := os.WriteFile(filepath.Join(root, "Workbase", "mcps", "jiangnan-workbase.md"), []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}
	m2 := wrap(r.handleIdentity(ctx, callReq(nil))).mapOK(t)
	wb2, _ := m2["workbase"].(map[string]any)
	if wb2["name"] != "即时读取 Workbase" {
		t.Fatalf("live name = %v", wb2["name"])
	}
	if !strings.Contains(fmtString(wb2["description"]), "立刻可见") {
		t.Fatalf("live desc = %v", wb2["description"])
	}
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}

func policyPublic(v any) string {
	switch m := v.(type) {
	case map[string]string:
		return m["public"]
	case map[string]any:
		s, _ := m["public"].(string)
		return s
	default:
		return ""
	}
}
