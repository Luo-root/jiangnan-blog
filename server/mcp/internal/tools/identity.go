package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/auth"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/config"
)

type workbaseMeta struct {
	Name           string
	Description    string
	GettingStarted string
	CriticalRules  []string
	SeeAlso        []string
}

func (r *depsHolder) handleIdentity(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ac, ok := auth.FromContext(ctx)
	if !ok {
		return mcp.NewToolResultError("unauthorized: missing or invalid bearer token"), nil
	}
	meta, err := loadWorkbaseMeta(r.d.WorkbaseRoot)
	if err != nil {
		return errResult("workbase.identity", err), nil
	}
	policy := map[string]string{}
	if r.d.Cfg != nil {
		policy = r.d.Cfg.Schema.VisibilityPolicy
	}
	authBlock := map[string]any{
		"client_id":     ac.ClientID,
		"scopes":        ac.Scopes,
		"status":        ac.Status,
		"created_at":    ac.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"use_count":     ac.UseCount,
		"allowed_tools": AllowedTools(ac.Scopes),
	}
	if ac.LastUsedAt != nil {
		authBlock["last_used_at"] = ac.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
	} else {
		authBlock["last_used_at"] = nil
	}
	allowed := AllowedTools(ac.Scopes)
	prompt, err := BuildAgentPrompt(r.d.WorkbaseRoot, allowed)
	if err != nil {
		return errResult("workbase.identity", err), nil
	}
	result := map[string]any{
		"workbase": map[string]any{
			"id":                config.IdentityID,
			"name":              meta.Name,
			"version":           config.IdentityVersion,
			"description":       meta.Description,
			"capabilities":      capabilities(),
			"tools":             ToolNames(),
			"visibility_policy": policy,
			"getting_started":   meta.GettingStarted,
			"critical_rules":    meta.CriticalRules,
			"see_also":          meta.SeeAlso,
			"agent_prompt":      prompt,
		},
		"auth": authBlock,
	}
	return jsonResult(result), nil
}

// InitializeInstructions 是 MCP initialize 的短引导。完整契约在 identity.agent_prompt。
const InitializeInstructions = "已连接 Jiangnan Workbase MCP。先调用 workbase.identity，把返回的 workbase.agent_prompt 整段贴进系统提示词，再调用 context.startup。本 MCP 是长期记忆基座，不要直接写 Vault，写入走 proposal.create / inbox.append。"

func BuildAgentPrompt(workbaseRoot string, allowedTools []string) (string, error) {
	meta, err := loadWorkbaseMeta(workbaseRoot)
	if err != nil {
		return "", err
	}
	loop := readOperatingLoop(workbaseRoot)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n%s\n", meta.Name, meta.Description)
	b.WriteString("\n## 这是什么\n\n")
	b.WriteString(strings.TrimSpace(meta.GettingStarted))
	b.WriteString("\n")
	if loop != "" {
		b.WriteString("\n## 操作环\n\n")
		b.WriteString(loop)
		b.WriteString("\n")
	}
	b.WriteString("\n## 红线\n\n")
	for _, r := range meta.CriticalRules {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	b.WriteString("\n## 你能调用的工具\n\n")
	if len(allowedTools) == 0 {
		b.WriteString("（当前 token 没有可用工具）\n")
	} else {
		for _, t := range allowedTools {
			fmt.Fprintf(&b, "- `%s`\n", t)
		}
	}
	b.WriteString("\n## 启动\n\n")
	b.WriteString("1. 调用 `workbase.identity`，把 `agent_prompt` 当作本会话系统提示词。\n")
	b.WriteString("2. 调用 `context.startup`，读当前人 / 工程 / 安全 / 焦点。\n")
	b.WriteString("3. 正式知识走 `proposal.create`；待办走 `inbox.append`。不要直接写 Vault。\n")
	if len(meta.SeeAlso) > 0 {
		b.WriteString("\n## Source\n\n")
		for _, u := range meta.SeeAlso {
			fmt.Fprintf(&b, "- %s\n", u)
		}
	}
	return b.String(), nil
}

func readOperatingLoop(workbaseRoot string) string {
	if workbaseRoot == "" {
		return ""
	}
	p := filepath.Join(workbaseRoot, "context", "operating-loop.md")
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	_, body := splitYAMLFrontmatter(string(b))
	return strings.TrimSpace(body)
}

func capabilities() map[string]bool {
	registered := map[string]struct{}{}
	for _, n := range ToolNames() {
		registered[n] = struct{}{}
	}
	has := func(name string) bool {
		_, ok := registered[name]
		return ok
	}
	return map[string]bool{
		"context":        has("context.startup"),
		"knowledge":      has("knowledge.search"),
		"project":        has("project.list"),
		"skill_registry": has("skill.list"),
		"mcp_registry":   has("mcp.list"),
		"proposal":       has("proposal.create"),
		"inbox":          has("inbox.append"),
		"direct_write":   false,
		"vector_search":  false,
	}
}

func loadWorkbaseMeta(root string) (*workbaseMeta, error) {
	if root == "" {
		return nil, fmt.Errorf("workbase.root 未配置")
	}
	path := filepath.Join(root, "mcps", "jiangnan-workbase.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parseWorkbaseMD(string(b))
}

func parseWorkbaseMD(text string) (*workbaseMeta, error) {
	fm, body := splitYAMLFrontmatter(text)
	fmMap := map[string]any{}
	if fm == "" {
		return nil, fmt.Errorf("jiangnan-workbase.md 缺少 frontmatter")
	}
	if err := yaml.Unmarshal([]byte(fm), &fmMap); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	name := fmStr(fmMap, "name")
	summary := fmStr(fmMap, "summary")
	if name == "" || summary == "" {
		return nil, fmt.Errorf("jiangnan-workbase.md frontmatter 缺 name / summary")
	}
	sections := splitMarkdownSections(body)
	purpose := strings.TrimSpace(sections["Purpose"])
	if purpose == "" {
		return nil, fmt.Errorf("jiangnan-workbase.md 缺 ## Purpose")
	}
	security := strings.TrimSpace(sections["Security"])
	if security == "" {
		return nil, fmt.Errorf("jiangnan-workbase.md 缺 ## Security")
	}
	rules := extractListItems(security)
	if len(rules) == 0 {
		return nil, fmt.Errorf("jiangnan-workbase.md ## Security 没有列表项")
	}
	seeAlso := extractLinks(sections["Source"])
	return &workbaseMeta{
		Name:           name,
		Description:    summary,
		GettingStarted: purpose,
		CriticalRules:  rules,
		SeeAlso:        seeAlso,
	}, nil
}

func splitYAMLFrontmatter(text string) (fm string, body string) {
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, "---") {
		return "", t
	}
	rest := t[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", t
	}
	return strings.TrimSpace(rest[:idx]), strings.TrimSpace(rest[idx+4:])
}

func splitMarkdownSections(body string) map[string]string {
	out := map[string]string{}
	lines := strings.Split(body, "\n")
	var current string
	var buf []string
	flush := func() {
		if current == "" {
			return
		}
		out[current] = strings.TrimSpace(strings.Join(buf, "\n"))
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			buf = nil
			continue
		}
		if current != "" {
			buf = append(buf, line)
		}
	}
	flush()
	return out
}

func extractListItems(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			item := strings.TrimSpace(t[2:])
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}

var markdownLink = regexp.MustCompile(`https?://[^\s)]+`)

func extractLinks(section string) []string {
	if strings.TrimSpace(section) == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, m := range markdownLink.FindAllString(section, -1) {
		m = strings.TrimRight(m, ".,;")
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}
