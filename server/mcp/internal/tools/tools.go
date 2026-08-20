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
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	"github.com/Luo-root/jiangnan-blog/mcp/internal/sanitize"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/vault"
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
		mcp.WithDescription("在授权知识库中搜索（正文全文 + 标题/标签/frontmatter/反链/热度/时效；intent=why/when/entity/general）。"),
		mcp.WithString("query", mcp.Required(), mcp.Description("搜索关键词。")),
		mcp.WithString("scope", mcp.Description("省略则全局搜索。")),
		mcp.WithString("intent", mcp.Description("weighs: why/when/entity/general（默认 general）。")),
		mcp.WithNumber("limit", mcp.Description("最大返回数（默认 10）。")),
	), r.handleKnowledgeSearch)

	srv.AddTool(mcp.NewTool("knowledge.get",
		mcp.WithDescription("读取单篇笔记/articles/context 的正文、frontmatter 与 WikiLink。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("笔记标识（相对路径或文件名）。")),
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
		mcp.WithDescription("创建统一写入提案（需审批后才 apply）。"),
		mcp.WithString("kind", mcp.Required(), mcp.Description("提案类型（如 context_update）。")),
		mcp.WithString("reason", mcp.Required(), mcp.Description("写入理由。")),
		mcp.WithString("content", mcp.Required(), mcp.Description("提案正文（Markdown）。")),
		mcp.WithString("created_by", mcp.Description("创建者标识（默认 unknown）。")),
		mcp.WithString("target_type", mcp.Description("目标类型。")),
		mcp.WithString("target_id", mcp.Description("目标 id。")),
		mcp.WithString("target_path", mcp.Description("目标路径。")),
		mcp.WithString("op_type", mcp.Description("操作类型（如 patch_section）。")),
		mcp.WithString("op_section", mcp.Description("操作的 section 名。")),
	), r.handleProposalCreate)

	srv.AddTool(mcp.NewTool("proposal.list",
		mcp.WithDescription("列出所有已创建的提案（摘要）：pending/approved/applied/rejected/conflict。"),
	), r.handleProposalList)

	srv.AddTool(mcp.NewTool("proposal.get",
		mcp.WithDescription("读取单条提案的完整内容（含 diff/preview 和 receipt）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("提案 id。")),
	), r.handleProposalGet)

	srv.AddTool(mcp.NewTool("inbox.append",
		mcp.WithDescription("创建一条待办（状态 pending），只存 VPS 私有区，不进本地 Vault。"),
		mcp.WithString("content", mcp.Required(), mcp.Description("待办正文（Markdown）。")),
		mcp.WithString("created_by", mcp.Description("创建者（默认 mcp_client）。")),
	), r.handleInboxAppend)

	srv.AddTool(mcp.NewTool("inbox.update",
		mcp.WithDescription("编辑待办内容或变更状态（pending/reviewing/done/abandoned）。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("待办标识。")),
		mcp.WithString("status", mcp.Description("新状态（pending|reviewing|done|abandoned），省略则只改内容。")),
		mcp.WithString("content", mcp.Description("替换正文，省略则保留原内容。")),
	), r.handleInboxUpdate)

	srv.AddTool(mcp.NewTool("inbox.list",
		mcp.WithDescription("列出所有待办摘要。"),
	), r.handleInboxList)

	srv.AddTool(mcp.NewTool("inbox.get",
		mcp.WithDescription("读取单条待办的完整内容。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("待办标识。")),
	), r.handleInboxGet)

	srv.AddTool(mcp.NewTool("audit.list_recent",
		mcp.WithDescription("查看最近工具调用审计（detail=含目标 id, hashed=含内容 SHA-256）。"),
		mcp.WithString("mode", mcp.Description("detail | hashed，默认 detail。")),
		mcp.WithNumber("limit", mcp.Description("返回条数（默认 20，最大 200）。")),
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

func checkSensitive(texts ...string) error {
	for _, t := range texts {
		if hits := sanitize.FindSensitive(t); len(hits) > 0 {
			return &SensitiveError{Patterns: hits}
		}
	}
	return nil
}

type SensitiveError struct{ Patterns []string }

func (e *SensitiveError) Error() string {
	// 旧实现：命中当错误拒绝。契约已改成只 warning、不拒绝。
	// 重构前不要把这行当规格。见 SCHEMA.md §21。
	return "prohibited: 内容命中了敏感模式，请手动脱敏后重提：" + strings.Join(e.Patterns, "、")
}

// readBody 读文件并去掉 frontmatter，仅返回正文。
func readBody(abs string) string {
	b, _ := os.ReadFile(abs)
	return stripFrontmatter(string(b))
}

func stripFrontmatter(text string) string {
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, "---") {
		return t
	}
	rest := t[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return t
	}
	return strings.TrimSpace(rest[idx+4:])
}

// idMatches 完整 id 匹配，同时支持只给文件名（去扩展名）匹配。
func idMatches(candidate, want string) bool {
	if candidate == want {
		return true
	}
	i := strings.LastIndexByte(candidate, '/')
	if i >= 0 && candidate[i+1:] == want {
		return true
	}
	return candidate == want
}

// findNoteByID 在 notes 切片中按 id 查找。
func findNoteByID(notes []vault.Note, id string) *vault.Note {
	for i := range notes {
		if n := &notes[i]; idMatches(n.ID, id) {
			return n
		}
	}
	return nil
}

// findContextByID 在 context 切片中按 id 查找。
func findContextByID(packs []vault.ContextPack, id string) *vault.ContextPack {
	for i := range packs {
		if c := &packs[i]; idMatches(c.ID, id) {
			return c
		}
	}
	return nil
}

// noteType 从 section / FM 推导类型。
func noteType(n vault.Note) string {
	if n.Section == "项目" {
		return "project"
	}
	if n.Section == "Workbase" {
		switch fmStr(n.FM, "kind") {
		case "skill":
			return "skill"
		case "mcp_server":
			return "mcp_server"
		}
		if fmStr(n.FM, "type") == "context_pack" {
			return "context_pack"
		}
	}
	if n.Section == "文章" {
		return "article"
	}
	return "note"
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

func jsonString(v interface{}) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return s
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// concatStartupPacks 返回 startup=true 的 context pack 拼接正文（按 priority）。
func concatStartupPacks(packs []vault.ContextPack) []vault.ContextPack {
	var out []vault.ContextPack
	for i := range packs {
		if packs[i].Startup {
			out = append(out, packs[i])
		}
	}
	// high > medium > low
	prio := map[string]int{"high": 3, "medium": 2, "low": 1, "": 0}
	sort.SliceStable(out, func(i, j int) bool { return prio[out[i].Priority] > prio[out[j].Priority] })
	return out
}

// extractWikiLinks 从 markdown 文本中提取 [[target]]。
func extractWikiLinks(text string) []string {
	var out []string
	seen := map[string]bool{}
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], "[[")
		if j < 0 {
			break
		}
		i += j + 2
		end := strings.Index(text[i:], "]]")
		if end < 0 {
			break
		}
		target := strings.TrimSpace(text[i : i+end])
		if idx := strings.Index(target, "|"); idx >= 0 {
			target = strings.TrimSpace(target[:idx])
		}
		if target != "" && !seen[target] {
			seen[target] = true
			out = append(out, target)
		}
		i += end + 2
	}
	return out
}

type searchHit struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	PathHint      string             `json:"path_hint"`
	Type          string             `json:"type"`
	Visibility    string             `json:"visibility"`
	Summary       string             `json:"summary"`
	MatchedFields []string           `json:"matched_fields"`
	Score         float64            `json:"score"`
	MatchedVia    string             `json:"matched_via"`
	Signals       map[string]float64 `json:"signals,omitempty"`
}

func findString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// tokenize 把中文/英文 query 拆成小写 token。
func tokenize(q string) []string {
	// 简单：按空格/标点拆
	var out []string
	var buf strings.Builder
	for _, r := range q {
		if strings.ContainsRune(" \t,，。.!！?", r) {
			if buf.Len() > 0 {
				out = append(out, strings.ToLower(buf.String()))
				buf.Reset()
			}
		} else {
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		out = append(out, strings.ToLower(buf.String()))
	}
	return out
}

func anyHow(s string, tokens []string) int {
	for _, tk := range tokens {
		if tk != "" && strings.Contains(strings.ToLower(s), tk) {
			return 1
		}
	}
	return 0
}

// baseCommit 读取 Vault git base commit。
func baseCommit(gitDir string) string {
	if gitDir == "" {
		return ""
	}
	cmd := exec.Command("git", "--git-dir="+gitDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// --------------------------------------------------------------------------
// 工具处理器
// --------------------------------------------------------------------------

func (r *depsHolder) handleContextStartup(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packs := concatStartupPacks(r.d.Idx.ContextPacks())
	sections := make([]map[string]string, 0, len(packs))
	var content strings.Builder
	for _, c := range packs {
		sections = append(sections, map[string]string{"id": c.ID, "title": c.Title, "priority": c.Priority})
		content.WriteString(stripFrontmatter(c.Content))
		content.WriteString("\n\n")
		r.d.Idx.Hit(c.ID)
	}
	result := map[string]any{
		"packs":   sections,
		"content": content.String(),
	}
	return jsonResult(result), nil
}

func (r *depsHolder) handleContextGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	c := findContextByID(r.d.Idx.ContextPacks(), id)
	if c == nil {
		return mcp.NewToolResultError("context_pack not found: " + id), nil
	}
	r.d.Idx.Hit(c.ID)
	result := map[string]any{
		"id":         c.ID,
		"title":      c.Title,
		"visibility": c.Visibility,
		"updated_at": c.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
		"content":    stripFrontmatter(c.Content),
		"metadata":   c.Priority,
	}
	return jsonResult(result), nil
}

func (r *depsHolder) handleKnowledgeSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("required argument query missing"), nil
	}
	intent := req.GetString("intent", "general")
	limit := req.GetInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	tokens := tokenize(query)
	notes := r.d.Idx.Notes()
	now := time.Now()

	var hits []searchHit
	for _, n := range notes {
		// visibility:secret 默认不返回
		if n.Visibility == "secret" {
			continue
		}

		// 读正文做全文搜索
		bodyText := vault.ReadBody(n.Path)
		fmJSON := jsonString(n.FM)
		tagsStr := strings.Join(n.Tags, " ")

		titleScore := float64(anyHow(n.Title, tokens) * 3)
		tagsScore := float64(anyHow(tagsStr, tokens) * 2)
		fmScore := float64(anyHow(fmJSON, tokens))
		sectionScore := float64(anyHow(n.Section, tokens))
		bodyScore := float64(anyHow(bodyText, tokens) * 2) // fulltext
		accessScore := float64(r.d.Idx.Count(n.ID)) * 0.1

		// recency: 30 天内更新 +0.2
		recencyScore := 0.0
		if now.Sub(n.UpdatedAt).Hours() < 720 {
			recencyScore = 0.2
		}

		// wikilink_backref
		backlinks := r.d.Idx.Backlinks(n.ID)
		backrefScore := float64(len(backlinks)) * 0.5

		signals := map[string]float64{
			"title":            titleScore,
			"tags":             tagsScore,
			"frontmatter":      fmScore,
			"section":          sectionScore,
			"fulltext":         bodyScore,
			"wikilink_backref": backrefScore,
			"access":           accessScore,
			"recency":          recencyScore,
		}

		// intent 权重调整
		switch intent {
		case "why":
			signals["wikilink_backref"] *= 2.0
		case "when":
			signals["frontmatter"] *= 2.0
			signals["recency"] *= 2.0
		case "entity":
			signals["title"] *= 2.0
			signals["tags"] *= 2.0
		}

		// 查询命中门禁：title/tags/frontmatter/section/fulltext 至少一个命中才入结果
		// wikilink_backref/access/recency 只参与排序，不能单独让无关文档入榜
		queryMatch := titleScore + tagsScore + fmScore + sectionScore + bodyScore
		if queryMatch <= 0 {
			continue
		}

		score := 0.0
		for _, v := range signals {
			score += v
		}

		// matched_fields
		fields := []string{}
		signalOrder := []string{"title", "tags", "frontmatter", "section", "fulltext", "wikilink_backref", "access", "recency"}
		for _, sn := range signalOrder {
			if val, ok := signals[sn]; ok && val > 0 {
				fields = append(fields, sn)
			}
		}

		// matched_via
		via := []string{}
		if bodyScore > 0 {
			via = append(via, "fulltext")
		}
		if backrefScore > 0 {
			via = append(via, "wikilink_backref")
		}
		if titleScore > 0 || tagsScore > 0 || fmScore > 0 || sectionScore > 0 {
			via = append(via, "frontmatter")
		}

		hits = append(hits, searchHit{
			ID:            n.ID,
			Title:         n.Title,
			PathHint:      n.Section + "/" + path.Base(n.ID),
			Type:          noteType(n),
			Visibility:    n.Visibility,
			Summary:       n.Title,
			MatchedFields: fields,
			Score:         score,
			MatchedVia:    strings.Join(via, " + "),
			Signals:       signals,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return jsonResult(map[string]any{"results": hits}), nil
}

func (r *depsHolder) handleKnowledgeGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	n := findNoteByID(r.d.Idx.Notes(), id)
	if n == nil {
		return mcp.NewToolResultError("note not found: " + id), nil
	}
	// visibility:secret 默认不返回（§8 可见性模型）
	if n.Visibility == "secret" {
		return mcp.NewToolResultError("note not found: " + id), nil
	}
	r.d.Idx.Hit(n.ID)
	body := readBody(n.Path)
	// 敏感内容检测（§20.1：禁止泄露私钥/token/密码）
	if hits := sanitize.FindSensitive(body); len(hits) > 0 {
		return mcp.NewToolResultError("content blocked: sensitive patterns detected (" + strings.Join(hits, ", ") + ")"), nil
	}
	fwd := extractWikiLinks(body)
	backlinks := r.d.Idx.Backlinks(n.ID)
	result := map[string]any{
		"id":          n.ID,
		"title":       n.Title,
		"type":        noteType(*n),
		"visibility":  n.Visibility,
		"frontmatter": n.FM,
		"content":     body,
		"links": map[string]any{
			"forward":   fwd,
			"backlinks": backlinks,
		},
		"base_commit": baseCommit(r.d.GitDir),
	}
	return jsonResult(result), nil
}

func (r *depsHolder) handleProjectList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projs := r.d.Idx.Projects()
	out := make([]map[string]any, 0, len(projs))
	for _, p := range projs {
		out = append(out, map[string]any{
			"id":      p.ID,
			"name":    p.Name,
			"summary": p.Summary,
			"status":  p.Status,
			"tags":    p.Tags,
		})
	}
	return jsonResult(map[string]any{"projects": out}), nil
}

func (r *depsHolder) handleProjectGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	projs := r.d.Idx.Projects()
	for i := range projs {
		p := &projs[i]
		if idMatches(p.ID, id) {
			r.d.Idx.Hit(p.ID)
			return jsonResult(map[string]any{
				"id":          p.ID,
				"name":        p.Name,
				"summary":     p.Summary,
				"status":      p.Status,
				"tags":        p.Tags,
				"visibility":  p.Visibility,
				"content":     readBody(p.Path),
				"frontmatter": p.FM,
			}), nil
		}
	}
	return mcp.NewToolResultError("project not found: " + id), nil
}

func (r *depsHolder) handleSkillList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skills := r.d.Idx.Skills()
	out := make([]map[string]any, 0, len(skills))
	for _, s := range skills {
		out = append(out, map[string]any{
			"id":         s.ID,
			"name":       fmStr(s.FM, "name"),
			"summary":    fmStr(s.FM, "summary"),
			"tags":       s.Tags,
			"visibility": s.Visibility,
			"risk":       fmStr(s.FM, "risk"),
			"source":     s.Source,
		})
	}
	return jsonResult(map[string]any{"skills": out}), nil
}

func (r *depsHolder) handleSkillGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	skills := r.d.Idx.Skills()
	for i := range skills {
		s := &skills[i]
		if idMatches(s.ID, id) {
			r.d.Idx.Hit(s.ID)
			return jsonResult(map[string]any{
				"id":           s.ID,
				"name":         fmStr(s.FM, "name"),
				"summary":      fmStr(s.FM, "summary"),
				"visibility":   s.Visibility,
				"risk":         fmStr(s.FM, "risk"),
				"source":       s.Source,
				"content_type": "text/markdown",
				"content":      readBody(s.Path),
			}), nil
		}
	}
	return mcp.NewToolResultError("skill not found: " + id), nil
}

func (r *depsHolder) handleMCPList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ms := r.d.Idx.MCPServers()
	out := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, map[string]any{
			"id":            m.ID,
			"name":          fmStr(m.FM, "name"),
			"summary":       fmStr(m.FM, "summary"),
			"transport":     fmStr(m.FM, "transport"),
			"endpoint_hint": fmStr(m.FM, "endpoint"),
			"auth":          m.Auth,
			"visibility":    m.Visibility,
			"risk":          fmStr(m.FM, "risk"),
		})
	}
	return jsonResult(map[string]any{"servers": out}), nil
}

func (r *depsHolder) handleMCPGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	ms := r.d.Idx.MCPServers()
	for i := range ms {
		m := &ms[i]
		if idMatches(m.ID, id) {
			r.d.Idx.Hit(m.ID)
			scopes := []string{}
			if s, ok := m.FM["scopes"]; ok {
				switch v := s.(type) {
				case []interface{}:
					for _, sc := range v {
						if str, ok := sc.(string); ok {
							scopes = append(scopes, str)
						}
					}
				}
			}
			return jsonResult(map[string]any{
				"id":        m.ID,
				"transport": fmStr(m.FM, "transport"),
				"endpoint":  fmStr(m.FM, "endpoint"),
				"auth":      m.Auth,
				"scopes":    scopes,
				"source":    map[string]string{"repo": "https://github.com/Luo-root/jiangnan-blog"},
				"content":   readBody(m.Path),
			}), nil
		}
	}
	return mcp.NewToolResultError("mcp server not found: " + id), nil
}

func (r *depsHolder) handleProposalCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	kind := req.GetString("kind", "")
	reason := req.GetString("reason", "")
	content := req.GetString("content", "")
	createdBy := req.GetString("created_by", "unknown")

	if kind == "" || reason == "" || content == "" {
		return mcp.NewToolResultError("required arguments: kind, reason, content"), nil
	}
	if err := checkSensitive(content, reason); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	p := proposal.Proposal{
		Kind:       kind,
		Reason:     reason,
		BaseCommit: baseCommit(r.d.GitDir),
		Target: proposal.Target{
			Type: req.GetString("target_type", ""),
			ID:   req.GetString("target_id", ""),
			Path: req.GetString("target_path", ""),
		},
		Operation: proposal.Operation{
			Type:    req.GetString("op_type", ""),
			Section: req.GetString("op_section", ""),
		},
		Payload: proposal.Payload{
			Format:  "markdown",
			Content: content,
		},
		CreatedBy: createdBy,
	}
	created, err := r.d.Proposal.Create(p)
	if err != nil {
		return errResult("create proposal", err), nil
	}
	return jsonResult(map[string]any{
		"id":     created.ID,
		"status": created.Status,
	}), nil
}

func (r *depsHolder) handleProposalList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	props, err := r.d.Proposal.List()
	if err != nil {
		return errResult("list proposals", err), nil
	}
	type item struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Status    string `json:"status"`
		Reason    string `json:"reason"`
		CreatedBy string `json:"created_by"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]item, 0, len(props))
	for _, p := range props {
		out = append(out, item{
			ID:        p.ID,
			Kind:      p.Kind,
			Status:    string(p.Status),
			Reason:    p.Reason,
			CreatedBy: p.CreatedBy,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return jsonResult(map[string]any{"proposals": out}), nil
}

func (r *depsHolder) handleProposalGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	p, err := r.d.Proposal.Get(id)
	if err != nil {
		return errResult("get proposal", err), nil
	}
	if p == nil {
		return mcp.NewToolResultError("proposal not found: " + id), nil
	}

	// 生成 preview/diff（§9.13：proposal.get 返回 diff/preview 和 receipt）
	result := map[string]any{
		"id":          p.ID,
		"kind":        p.Kind,
		"status":      p.Status,
		"reason":      p.Reason,
		"created_by":  p.CreatedBy,
		"created_at":  p.CreatedAt,
		"base_commit": p.BaseCommit,
		"target":      p.Target,
		"operation":   p.Operation,
		"payload":     p.Payload,
		"preview":     proposalPreview(p, r.d.VaultRoot),
	}
	if p.Receipt != nil {
		result["receipt"] = p.Receipt
	}
	return jsonResult(result), nil
}

// proposalPreview 生成 proposal 的 before/after 预览（不实际写入）。
func proposalPreview(p *proposal.Proposal, vaultRoot string) map[string]any {
	if p.Target.Path == "" {
		return map[string]any{"mode": "create_file", "after": p.Payload.Content}
	}
	absPath := filepath.Join(vaultRoot, filepath.FromSlash(p.Target.Path))
	existing, err := os.ReadFile(absPath)
	if err != nil {
		return map[string]any{"mode": p.Operation.Type, "before": "(file does not exist)", "after": p.Payload.Content}
	}
	before := string(existing)
	preview := map[string]any{
		"mode":   p.Operation.Type,
		"before": truncatePreview(before, 500),
	}
	switch p.Operation.Type {
	case "append":
		preview["after"] = truncatePreview(before+"\n"+p.Payload.Content, 500)
	case "append_section":
		preview["after"] = truncatePreview(before+"\n"+p.Payload.Content, 500)
	case "patch_section":
		preview["after"] = p.Payload.Content
		preview["section"] = p.Operation.Section
	case "replace_frontmatter":
		preview["after"] = p.Payload.Content
		preview["section"] = "frontmatter"
	default:
		preview["after"] = truncatePreview(before+"\n"+p.Payload.Content, 500)
	}
	return preview
}

func truncatePreview(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (r *depsHolder) handleInboxAppend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	createdBy := req.GetString("created_by", "mcp_client")
	if content == "" {
		return mcp.NewToolResultError("required argument content missing"), nil
	}
	if err := checkSensitive(content); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	id, err := r.d.Inbox.Append(createdBy, content)
	if err != nil {
		return errResult("inbox append", err), nil
	}
	return jsonResult(map[string]any{"id": id, "status": "pending"}), nil
}

func (r *depsHolder) handleInboxUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	status := inbox.Status(req.GetString("status", ""))
	content := req.GetString("content", "")
	if content != "" {
		if err := checkSensitive(content); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	if err := r.d.Inbox.Update(id, status, content); err != nil {
		return errResult("inbox update", err), nil
	}
	return jsonResult(map[string]any{"id": id, "status": string(status)}), nil
}

func (r *depsHolder) handleInboxList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	items, err := r.d.Inbox.List()
	if err != nil {
		return errResult("inbox list", err), nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"id":         it.ID,
			"created_at": it.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at": it.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"created_by": it.CreatedBy,
			"summary":    it.Summary,
			"status":     it.Status,
		})
	}
	return jsonResult(map[string]any{"items": out}), nil
}

func (r *depsHolder) handleInboxGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	item, err := r.d.Inbox.Get(id)
	if err != nil {
		return errResult("inbox get", err), nil
	}
	if item == nil {
		return mcp.NewToolResultError("inbox item not found: " + id), nil
	}
	return jsonResult(map[string]any{
		"id":      item.ID,
		"status":  item.Status,
		"content": item.Content,
	}), nil
}

func (r *depsHolder) handleAuditListRecent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mode := req.GetString("mode", "detail")
	limit := req.GetInt("limit", 20)
	if limit > 200 {
		limit = 200
	}
	entries := r.d.Audit.List(mode, limit)
	type out struct {
		Time        string `json:"time"`
		Op          string `json:"op"`
		Scope       string `json:"scope"`
		ClientID    string `json:"client_id,omitempty"`
		TargetID    string `json:"target_id,omitempty"`
		ContentHash string `json:"content_hash,omitempty"`
	}
	result := make([]out, 0, len(entries))
	for _, e := range entries {
		result = append(result, out{
			Time:        e.Time.Format("2006-01-02T15:04:05Z07:00"),
			Op:          e.Op,
			Scope:       e.Scope,
			ClientID:    e.ClientID,
			TargetID:    e.TargetID,
			ContentHash: e.ContentHash,
		})
	}
	return jsonResult(map[string]any{"entries": result}), nil
}
