// Package tools 将 Agent Workbase 的全部能力注册为 MCP tools。
//
// 依赖 mcp-go server.ToolHandlerFunc 语义：
// handler(ctx, mcp.CallToolRequest) → (*mcp.CallToolResult, error)
//
// 工具按 scope 分组，供 auth 中间件做按权限过滤。
package tools

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/audit"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/config"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/inbox"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/index"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

// --------------------------------------------------------------------------
// scope 常量 — 供中间件使用
// --------------------------------------------------------------------------

const (
	ScopeReadContext   = "read:context"
	ScopeReadKnowledge = "read:knowledge"
	ScopeReadProject   = "read:project"
	ScopeReadRegistry  = "read:registry"
	ScopeWriteProposal = "write:proposal"
	ScopeReadInbox     = "read:inbox"
	ScopeWriteInbox    = "write:inbox"
	ScopeAudit         = "ops:audit"
)

// RequiredScope 返回工具名对应的 scope。空串 = 任意有效 token。
func RequiredScope(toolName string) string { return toolScopes[toolName] }

var toolScopes = map[string]string{
	"workbase.identity": "",
	"context.startup":   ScopeReadContext,
	"context.get":       ScopeReadContext,
	"knowledge.search":  ScopeReadKnowledge,
	"knowledge.get":     ScopeReadKnowledge,
	"project.list":      ScopeReadProject,
	"project.get":       ScopeReadProject,
	"skill.list":        ScopeReadRegistry,
	"skill.get":         ScopeReadRegistry,
	"mcp.list":          ScopeReadRegistry,
	"mcp.get":           ScopeReadRegistry,
	"proposal.create":   ScopeWriteProposal,
	"proposal.list":     ScopeWriteProposal,
	"proposal.get":      ScopeWriteProposal,
	"proposal.update":   ScopeWriteProposal,
	"inbox.append":      ScopeWriteInbox,
	"inbox.update":      ScopeWriteInbox,
	"inbox.list":        ScopeReadInbox,
	"inbox.get":         ScopeReadInbox,
	"audit.list_recent": ScopeAudit,
}

func ToolNames() []string {
	keys := make([]string, 0, len(toolScopes))
	for k := range toolScopes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func AllowedTools(scopes []string) []string {
	have := map[string]struct{}{}
	for _, s := range scopes {
		have[s] = struct{}{}
	}
	var out []string
	for _, name := range ToolNames() {
		need := toolScopes[name]
		if need == "" {
			out = append(out, name)
			continue
		}
		if _, ok := have[need]; ok {
			out = append(out, name)
		}
	}
	return out
}

// --------------------------------------------------------------------------
// Deps 聚合 Registry 所需外部依赖。
// --------------------------------------------------------------------------

type Deps struct {
	Idx          *index.Store
	Inbox        *inbox.Store
	Proposal     *proposal.Store
	Audit        *audit.Store
	Cfg          *config.Config
	VaultRoot    string
	WorkbaseRoot string
	GitDir       string
}

// Register 将所有工具注册到 mcp-go server。
func Register(srv *server.MCPServer, d Deps) {
	r := &depsHolder{d}

	srv.AddTool(mcp.NewTool("workbase.identity",
		mcp.WithDescription("返回 Workbase 自描述 + 当前 token 元数据。描述性字段从 vault 即时读。"),
	), r.handleIdentity)

	srv.AddTool(mcp.NewTool("context.startup",
		mcp.WithDescription("派生启动上下文：从 startup context packs 生成 Agent 快速入门摘要。"),
	), r.handleContextStartup)

	srv.AddTool(mcp.NewTool("context.get",
		mcp.WithDescription("读取单个 context pack 的完整内容。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("context pack 标识（文件名或完整相对路径，如 engineering-style）。")),
	), r.handleContextGet)

	srv.AddTool(mcp.NewTool("knowledge.search",
		mcp.WithDescription("在授权知识库中搜索 note / article。intent=why/when/entity/general；scope=all/public/private。"),
		mcp.WithString("query", mcp.Required(), mcp.Description("搜索关键词。")),
		mcp.WithString("scope", mcp.Description("all / public / private。省略 = all（含 draft）。secret 永不进 search。")),
		mcp.WithString("intent", mcp.Description("why / when / entity / general。省略 = general。非法值报 invalid_argument。")),
		mcp.WithArray("kind", mcp.Description("只认 note / article。缺省才用 [note, article]。过滤后为空 → 空结果。")),
		mcp.WithNumber("limit", mcp.Description("最大返回数（默认 10）。")),
	), r.handleKnowledgeSearch)

	srv.AddTool(mcp.NewTool("knowledge.get",
		mcp.WithDescription("读取单篇 note / article 的正文、frontmatter 与 WikiLink。id = vault 相对路径（含 .md）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("notes.id：vault 相对路径，正斜杠，含 .md。")),
	), r.handleKnowledgeGet)

	srv.AddTool(mcp.NewTool("project.list",
		mcp.WithDescription("列出所有项目的摘要（id/name/summary/status/tags）。"),
	), r.handleProjectList)

	srv.AddTool(mcp.NewTool("project.get",
		mcp.WithDescription("读取单个项目的完整上下文。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("项目标识。")),
	), r.handleProjectGet)

	srv.AddTool(mcp.NewTool("skill.list",
		mcp.WithDescription("列出 Skill Registry 中所有技能的摘要。"),
	), r.handleSkillList)

	srv.AddTool(mcp.NewTool("skill.get",
		mcp.WithDescription("返回 Skill 完整定义（正文 + 元信息，不做安装）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("skill 标识。")),
	), r.handleSkillGet)

	srv.AddTool(mcp.NewTool("mcp.list",
		mcp.WithDescription("列出 MCP Registry 中所有 MCP 服务器的摘要。"),
	), r.handleMCPList)

	srv.AddTool(mcp.NewTool("mcp.get",
		mcp.WithDescription("返回 MCP Server 完整定义（transport/auth/scopes + 正文）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("mcp 标识。")),
	), r.handleMCPGet)

	srv.AddTool(mcp.NewTool("proposal.create",
		mcp.WithDescription("创建统一写入提案（需审批后才 apply）。不要传 kind / validation.checks。"),
		mcp.WithString("expected_base", mcp.Description("刚读到的 vault HEAD。有传但对不上 → stale_base。")),
		mcp.WithObject("target", mcp.Required(), mcp.Description("写入目标。"), mcp.Properties(map[string]any{
			"type": map[string]any{"type": "string", "description": "note / context_pack / project / article / skill / mcp_server"},
			"id":   map[string]any{"type": "string", "description": "已存在条目的对外 id"},
			"path": map[string]any{"type": "string", "description": "vault 相对路径，入库前 ToSlash"},
		})),
		mcp.WithObject("operation", mcp.Required(), mcp.Description("写入操作。"), mcp.Properties(map[string]any{
			"type":    map[string]any{"type": "string", "description": "create_file / append / append_section / patch_section / register_item"},
			"section": map[string]any{"type": "string", "description": "append_section / patch_section 必填"},
		})),
		mcp.WithObject("payload", mcp.Required(), mcp.Description("写入内容。"), mcp.Properties(map[string]any{
			"format":  map[string]any{"type": "string", "description": "固定 markdown"},
			"content": map[string]any{"type": "string", "description": "内容"},
		})),
		mcp.WithString("reason", mcp.Description("创建原因，审计用。")),
		mcp.WithObject("risk", mcp.Description("风险标注。"), mcp.Properties(map[string]any{
			"level":   map[string]any{"type": "string", "description": "low / medium / high"},
			"reasons": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		})),
	), r.handleProposalCreate)

	srv.AddTool(mcp.NewTool("proposal.list",
		mcp.WithDescription("列出已创建的提案摘要。"),
		mcp.WithString("status", mcp.Description("pending / approved / applied / rejected / conflict / all")),
		mcp.WithString("created_by", mcp.Description("客户端 id 过滤")),
		mcp.WithString("since", mcp.Description("RFC3339 创建时间下限")),
		mcp.WithNumber("limit", mcp.Description("返回条数，默认 50")),
	), r.handleProposalList)

	srv.AddTool(mcp.NewTool("proposal.get",
		mcp.WithDescription("读取单条提案的完整内容（含 diff 和 receipt）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("提案 id。")),
	), r.handleProposalGet)

	srv.AddTool(mcp.NewTool("proposal.update",
		mcp.WithDescription("改尚未落盘的提案。只允许 pending / conflict；可改字段并追加评论。评论不改 status。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("提案 id。")),
		mcp.WithString("reason", mcp.Description("修改原因。")),
		mcp.WithObject("target", mcp.Description("改目标。给了就整段替换。"), mcp.Properties(map[string]any{
			"type": map[string]any{"type": "string"},
			"id":   map[string]any{"type": "string"},
			"path": map[string]any{"type": "string"},
		})),
		mcp.WithObject("operation", mcp.Description("改操作。给了就整段替换。"), mcp.Properties(map[string]any{
			"type":    map[string]any{"type": "string"},
			"section": map[string]any{"type": "string"},
		})),
		mcp.WithObject("payload", mcp.Description("改内容。给了就整段替换。"), mcp.Properties(map[string]any{
			"format":  map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		})),
		mcp.WithObject("comment", mcp.Description("追加一条评论。有 body 就 append。"), mcp.Properties(map[string]any{
			"body":     map[string]any{"type": "string", "description": "Markdown 正文"},
			"reply_to": map[string]any{"type": "string", "description": "回复的评论 id"},
		})),
	), r.handleProposalUpdate)

	srv.AddTool(mcp.NewTool("inbox.append",
		mcp.WithDescription("创建一条待办（状态 pending），只存运行时私有区，不进 Vault。"),
		mcp.WithString("content", mcp.Required(), mcp.Description("待办正文（Markdown）。")),
		mcp.WithString("title", mcp.Description("简短标题。")),
		mcp.WithArray("tags", mcp.Description("标签。"), mcp.WithStringItems()),
	), r.handleInboxAppend)

	srv.AddTool(mcp.NewTool("inbox.update",
		mcp.WithDescription("编辑待办内容或变更状态（pending/reviewing/done/abandoned）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("待办标识。")),
		mcp.WithString("status", mcp.Description("新状态，省略则只改内容。")),
		mcp.WithString("content", mcp.Description("替换正文，省略则保留原内容。")),
		mcp.WithString("title", mcp.Description("修改标题。")),
		mcp.WithArray("tags", mcp.Description("替换标签。省略则保留。"), mcp.WithStringItems()),
		mcp.WithObject("comment", mcp.Description("追加一条评论。有 body 就 append。评论不改 status。"), mcp.Properties(map[string]any{
			"body":     map[string]any{"type": "string"},
			"reply_to": map[string]any{"type": "string"},
		})),
	), r.handleInboxUpdate)

	srv.AddTool(mcp.NewTool("inbox.list",
		mcp.WithDescription("列出所有待办摘要。"),
	), r.handleInboxList)

	srv.AddTool(mcp.NewTool("inbox.get",
		mcp.WithDescription("读取单条待办的完整内容。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("待办标识。")),
	), r.handleInboxGet)

	srv.AddTool(mcp.NewTool("audit.list_recent",
		mcp.WithDescription("查看最近工具调用审计。不返回 token 原文、args 原文。"),
		mcp.WithNumber("limit", mcp.Description("返回条数，默认 100。")),
		mcp.WithString("since", mcp.Description("RFC3339 时间下限。")),
		mcp.WithString("tool", mcp.Description("工具名过滤。")),
		mcp.WithString("client_id", mcp.Description("客户端过滤。")),
		mcp.WithString("result_status", mcp.Description("success / error / unauthorized / forbidden / all。省略 = all。")),
	), r.handleAuditListRecent)
}

// --------------------------------------------------------------------------
// 内部持有
// --------------------------------------------------------------------------

type depsHolder struct{ d Deps }

// --------------------------------------------------------------------------
// 辅助
// --------------------------------------------------------------------------

func jsonResult(v any) *mcp.CallToolResult {
	return mcp.NewToolResultStructuredOnly(v)
}

func errResult(msg string, err error) *mcp.CallToolResult {
	text := msg
	if err != nil {
		text = msg + ": " + err.Error()
	}
	return mcp.NewToolResultError(text)
}

func fmStr(fm map[string]interface{}, key string) string {
	v, ok := fm[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		b, _ := json.Marshal(v)
		return strings.Trim(string(b), `"`)
	}
}

func (r *depsHolder) handleAuditListRecent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := req.GetInt("limit", 0)
	sinceStr := req.GetString("since", "")
	tool := req.GetString("tool", "")
	clientID := req.GetString("client_id", "")
	status := req.GetString("result_status", "")
	switch status {
	case "", "all", "success", "error", "unauthorized", "forbidden":
	default:
		return mcp.NewToolResultError("invalid_argument: result_status"), nil
	}
	var since time.Time
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339Nano, sinceStr)
		if err != nil {
			t, err = time.Parse(time.RFC3339, sinceStr)
		}
		if err != nil {
			return mcp.NewToolResultError("invalid_argument: since"), nil
		}
		since = t
	}
	entries := r.d.Audit.List(audit.Filter{
		Limit:        limit,
		Since:        since,
		Tool:         tool,
		ClientID:     clientID,
		ResultStatus: status,
	})
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		item := map[string]any{
			"ts":            e.TS.UTC().Format(time.RFC3339Nano),
			"tool":          e.Tool,
			"client_id":     e.ClientID,
			"scopes":        e.Scopes,
			"args_digest":   e.ArgsDigest,
			"result_status": e.ResultStatus,
			"duration_ms":   e.DurationMS,
		}
		if e.Error != "" {
			item["error"] = e.Error
		}
		if e.TargetPath != "" {
			item["target_path"] = e.TargetPath
		}
		if e.Commit != "" {
			item["commit"] = e.Commit
		}
		if e.BaseCommit != "" {
			item["base_commit"] = e.BaseCommit
		}
		out = append(out, item)
	}
	return jsonResult(map[string]any{"entries": out}), nil
}
