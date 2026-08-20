package tools

import (
	"strings"
	"testing"
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

https://github.com/Luo-root/jiangnan-blog
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
	if len(meta.SeeAlso) != 1 || !strings.Contains(meta.SeeAlso[0], "jiangnan-blog") {
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
