package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/search"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/vault"
)

func (r *depsHolder) handleContextStartup(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packs := concatStartupPacks(r.d.Idx.ContextPacks())
	sections := make([]map[string]string, 0, len(packs))
	var content strings.Builder
	for i, c := range packs {
		if c.Visibility == "secret" {
			continue
		}
		sections = append(sections, map[string]string{
			"id":       vault.ToolID(c.Note),
			"title":    contextTitle(c),
			"priority": c.Priority,
		})
		if i > 0 {
			content.WriteString("\n\n")
		}
		content.WriteString(stripFrontmatter(c.Content))
	}
	return jsonResult(map[string]any{
		"packs":   sections,
		"content": content.String(),
	}), nil
}

func (r *depsHolder) handleContextGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	c := findTypedByToolID(r.d.Idx.ContextPacks(), id, func(p vault.ContextPack) vault.Note { return p.Note })
	if c == nil {
		return mcp.NewToolResultError("context_pack not found: " + id), nil
	}
	if c.Visibility == "secret" {
		return mcp.NewToolResultError("secret_blocked"), nil
	}
	r.d.Idx.Hit(c.ID)
	meta := map[string]any{}
	for k, v := range c.FM {
		meta[k] = v
	}
	return jsonResult(map[string]any{
		"id":         vault.ToolID(c.Note),
		"title":      contextTitle(*c),
		"visibility": c.Visibility,
		"updated_at": c.UpdatedAt.Format(time.RFC3339),
		"content":    stripFrontmatter(c.Content),
		"metadata":   meta,
	}), nil
}

func (r *depsHolder) handleKnowledgeSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("required argument query missing"), nil
	}

	args := req.GetArguments()
	intent := req.GetString("intent", "")
	if _, ok := args["intent"]; !ok {
		intent = "general"
	}
	switch intent {
	case "why", "when", "entity", "general":
	default:
		return mcp.NewToolResultError("invalid_argument: intent"), nil
	}

	scope := req.GetString("scope", "")
	if _, ok := args["scope"]; !ok {
		scope = "all"
	}
	switch scope {
	case "all", "public", "private":
	default:
		return mcp.NewToolResultError("invalid_argument: scope"), nil
	}

	kinds, kindsGiven := parseSearchKinds(args)
	limit := req.GetInt("limit", search.DefaultLimit)
	if limit <= 0 {
		limit = search.DefaultLimit
	}
	if limit > search.MaxLimit {
		limit = search.MaxLimit
	}

	if kindsGiven && len(kinds) == 0 {
		return jsonResult(emptySearch(query)), nil
	}

	weights := search.DefaultWeights()
	bias := search.DefaultIntentBias()
	halfLife := 7.0
	if r.d.Cfg != nil {
		weights = search.MergeWeights(r.d.Cfg.Knowledge.Search.Weights)
		bias = r.d.Cfg.Knowledge.Search.IntentBias
		if r.d.Cfg.Index.Access.HalfLifeDays > 0 {
			halfLife = r.d.Cfg.Index.Access.HalfLifeDays
		}
	}
	intentMul := search.IntentBias(bias, intent)
	tokens := search.Tokenize(query)
	now := time.Now()
	kindSet := map[string]struct{}{}
	for _, k := range kinds {
		kindSet[k] = struct{}{}
	}

	var hits []searchHit
	for _, n := range r.d.Idx.Notes() {
		if n.Kind != "note" && n.Kind != "article" {
			continue
		}
		if _, ok := kindSet[n.Kind]; !ok {
			continue
		}
		if !vault.MatchScope(n.Visibility, scope) {
			continue
		}

		fmJSON := jsonString(n.FM)
		tagsStr := strings.Join(n.Tags, " ")
		headings := strings.Join(n.Headings, "\n")
		titleHit := search.Hit(n.Title, tokens)
		tagsHit := search.Hit(tagsStr, tokens)
		fmHit := search.Hit(fmJSON, tokens)
		sectionHit := search.Hit(headings, tokens)
		bodyHit := search.Hit(n.Body, tokens)
		if !titleHit && !tagsHit && !fmHit && !sectionHit && !bodyHit {
			continue
		}

		w := func(name string) float64 {
			v := weights[name]
			if m, ok := intentMul[name]; ok && m != 0 {
				v *= m
			}
			return v
		}
		signals := map[string]float64{
			"title":            boolScore(titleHit, w("title")),
			"tags":             boolScore(tagsHit, w("tags")),
			"frontmatter":      boolScore(fmHit, w("frontmatter")),
			"section":          boolScore(sectionHit, w("section")),
			"fulltext":         boolScore(bodyHit, w("fulltext")),
			"wikilink_backref": float64(len(r.d.Idx.Backlinks(n.ID))) * w("wikilink_backref"),
			"access":           search.Access(r.d.Idx.Count(n.ID), r.d.Idx.LastAccess(n.ID), now, halfLife, w("access")),
			"recency":          search.Recency(n.UpdatedAt, now, w("recency")),
		}
		score := 0.0
		fields := []string{}
		via := []string{}
		for _, name := range []string{"title", "tags", "frontmatter", "section", "fulltext", "wikilink_backref", "access", "recency"} {
			if signals[name] > 0 {
				score += signals[name]
				fields = append(fields, name)
			}
		}
		if bodyHit {
			via = append(via, "fulltext")
		}
		if signals["wikilink_backref"] > 0 {
			via = append(via, "wikilink_backref")
		}
		if titleHit || tagsHit || fmHit || sectionHit {
			via = append(via, "frontmatter")
		}
		hits = append(hits, searchHit{
			ID:            n.ID,
			Title:         n.Title,
			PathHint:      n.ID,
			Kind:          n.Kind,
			Visibility:    n.Visibility,
			Summary:       n.Summary,
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
	if len(hits) == 0 {
		return jsonResult(emptySearch(query)), nil
	}
	return jsonResult(map[string]any{"results": hits}), nil
}

func (r *depsHolder) handleKnowledgeGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	n := r.d.Idx.NoteByID(id)
	if n == nil || (n.Kind != "note" && n.Kind != "article") {
		return mcp.NewToolResultError("note not found: " + vault.NormalizeID(id)), nil
	}
	if n.Visibility == "secret" {
		return mcp.NewToolResultError("secret_blocked"), nil
	}
	r.d.Idx.Hit(n.ID)
	fwd := linkObjects(r, r.d.Idx.ForwardLinks(n.ID), n.Body, true)
	back := linkObjects(r, backlinkEdges(r, n.ID), n.Body, false)
	return jsonResult(map[string]any{
		"id":            n.ID,
		"title":         n.Title,
		"path_hint":     n.ID,
		"kind":          n.Kind,
		"visibility":    n.Visibility,
		"updated_at":    n.UpdatedAt.Format(time.RFC3339),
		"body":          n.Body,
		"frontmatter":   n.FM,
		"forward_links": fwd,
		"backlinks":     back,
		"base_commit":   baseCommit(r.d.GitDir),
	}), nil
}

func (r *depsHolder) handleProjectList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projs := r.d.Idx.Projects()
	out := make([]map[string]any, 0, len(projs))
	for _, p := range projs {
		if !vault.VisibleInList(p.Visibility) {
			continue
		}
		item := map[string]any{
			"id":         vault.ToolID(p.Note),
			"name":       fmOr(p.FM, "name", p.Title),
			"summary":    p.Summary,
			"path_hint":  p.ID,
			"visibility": p.Visibility,
		}
		if s := fmStr(p.FM, "status"); s != "" {
			item["status"] = s
		}
		if v := p.FM["stack"]; v != nil {
			item["stack"] = v
		}
		if v := p.FM["links"]; v != nil {
			item["links"] = v
		}
		if s := fmStr(p.FM, "date"); s != "" {
			item["date"] = s
		}
		out = append(out, item)
	}
	return jsonResult(map[string]any{"projects": out}), nil
}

func (r *depsHolder) handleProjectGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	p := findTypedByToolID(r.d.Idx.Projects(), id, func(x vault.Project) vault.Note { return x.Note })
	if p == nil {
		return mcp.NewToolResultError("project not found: " + id), nil
	}
	if p.Visibility == "secret" {
		return mcp.NewToolResultError("secret_blocked"), nil
	}
	r.d.Idx.Hit(p.ID)
	body := p.Body
	result := map[string]any{
		"id":          vault.ToolID(p.Note),
		"name":        fmOr(p.FM, "name", p.Title),
		"summary":     p.Summary,
		"path_hint":   p.ID,
		"visibility":  p.Visibility,
		"body":        body,
		"frontmatter": p.FM,
	}
	if s := fmStr(p.FM, "status"); s != "" {
		result["status"] = s
	}
	if v := p.FM["stack"]; v != nil {
		result["stack"] = v
	}
	if v := p.FM["links"]; v != nil {
		result["links"] = v
	}
	if s := sectionText(body, "当前重点"); s != "" {
		result["current_focus"] = s
	}
	if items := extractListItems(sectionText(body, "下一步")); len(items) > 0 {
		result["next_steps"] = items
	}
	if items := extractListItems(sectionText(body, "决策")); len(items) > 0 {
		result["decisions"] = items
	}
	return jsonResult(result), nil
}

func (r *depsHolder) handleSkillList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skills := r.d.Idx.Skills()
	out := make([]map[string]any, 0, len(skills))
	for _, s := range skills {
		if !vault.VisibleInList(s.Visibility) {
			continue
		}
		item := map[string]any{
			"id":         vault.ToolID(s.Note),
			"name":       fmOr(s.FM, "name", s.Title),
			"summary":    s.Summary,
			"visibility": s.Visibility,
			"path_hint":  s.ID,
		}
		if s.Risk != "" {
			item["risk"] = s.Risk
		}
		if len(s.Tags) > 0 {
			item["tags"] = s.Tags
		}
		if s.Source != nil {
			item["source"] = s.Source
		}
		out = append(out, item)
	}
	return jsonResult(map[string]any{"skills": out}), nil
}

func (r *depsHolder) handleSkillGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	s := findTypedByToolID(r.d.Idx.Skills(), id, func(x vault.Skill) vault.Note { return x.Note })
	if s == nil {
		return mcp.NewToolResultError("skill not found: " + id), nil
	}
	if s.Visibility == "secret" {
		return mcp.NewToolResultError("secret_blocked"), nil
	}
	r.d.Idx.Hit(s.ID)
	result := map[string]any{
		"id":          vault.ToolID(s.Note),
		"name":        fmOr(s.FM, "name", s.Title),
		"summary":     s.Summary,
		"visibility":  s.Visibility,
		"path_hint":   s.ID,
		"body":        s.Body,
		"frontmatter": s.FM,
	}
	if s.Risk != "" {
		result["risk"] = s.Risk
	}
	if s.Source != nil {
		result["source"] = s.Source
	}
	if v := fmStr(s.FM, "license"); v != "" {
		result["license"] = v
	}
	if !s.UpdatedAt.IsZero() {
		result["updated_at"] = s.UpdatedAt.Format(time.RFC3339)
	}
	return jsonResult(result), nil
}

func (r *depsHolder) handleMCPList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ms := r.d.Idx.MCPServers()
	out := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		if !vault.VisibleInList(m.Visibility) {
			continue
		}
		authType := ""
		if auth, ok := m.Auth.(map[string]any); ok {
			authType, _ = auth["type"].(string)
		}
		item := map[string]any{
			"id":         vault.ToolID(m.Note),
			"name":       fmOr(m.FM, "name", m.Title),
			"summary":    m.Summary,
			"transport":  m.Transport,
			"auth_type":  authType,
			"visibility": m.Visibility,
			"path_hint":  m.ID,
		}
		if m.Risk != "" {
			item["risk"] = m.Risk
		}
		if v := m.FM["source"]; v != nil {
			item["source"] = v
		}
		out = append(out, item)
	}
	return jsonResult(map[string]any{"servers": out}), nil
}

func (r *depsHolder) handleMCPGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	m := findTypedByToolID(r.d.Idx.MCPServers(), id, func(x vault.MCPServer) vault.Note { return x.Note })
	if m == nil {
		return mcp.NewToolResultError("mcp server not found: " + id), nil
	}
	if m.Visibility == "secret" {
		return mcp.NewToolResultError("secret_blocked"), nil
	}
	r.d.Idx.Hit(m.ID)
	result := map[string]any{
		"id":          vault.ToolID(m.Note),
		"name":        fmOr(m.FM, "name", m.Title),
		"summary":     m.Summary,
		"transport":   m.Transport,
		"auth":        m.Auth,
		"visibility":  m.Visibility,
		"path_hint":   m.ID,
		"body":        m.Body,
		"frontmatter": m.FM,
	}
	if m.EndpointHint != "" {
		result["endpoint"] = m.EndpointHint
	}
	if m.Risk != "" {
		result["risk"] = m.Risk
	}
	if !m.UpdatedAt.IsZero() {
		result["updated_at"] = m.UpdatedAt.Format(time.RFC3339)
	}
	return jsonResult(result), nil
}

type searchHit struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	PathHint      string             `json:"path_hint"`
	Kind          string             `json:"kind"`
	Visibility    string             `json:"visibility"`
	Summary       string             `json:"summary"`
	MatchedFields []string           `json:"matched_fields"`
	Score         float64            `json:"score"`
	MatchedVia    string             `json:"matched_via"`
	Signals       map[string]float64 `json:"signals"`
}

func emptySearch(query string) map[string]any {
	return map[string]any{
		"results": []any{},
		"message": "未查询到相关内容",
		"suggestions": []string{
			"缩短关键词：去掉修饰词（'的'/'一个'/'关于'）",
			"改用更通用的词：例如 'kubernetes' 替代 'k8s pod 调度'",
			"检查 scope 权限：当前 token scope 是否包含 read:knowledge",
			"检查 visibility：public 内容只能搜到 public 知识",
		},
		"query_echo":       query,
		"executed_signals": []string{"title", "tags", "frontmatter", "section", "fulltext"},
	}
}

func parseSearchKinds(args map[string]any) (kinds []string, given bool) {
	if args == nil {
		return []string{"note", "article"}, false
	}
	raw, ok := args["kind"]
	if !ok {
		return []string{"note", "article"}, false
	}
	given = true
	var in []string
	switch v := raw.(type) {
	case []string:
		in = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				in = append(in, s)
			}
		}
	case string:
		if v != "" {
			in = []string{v}
		}
	}
	seen := map[string]struct{}{}
	for _, k := range in {
		if k == "note" || k == "article" {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			kinds = append(kinds, k)
		}
	}
	return kinds, true
}

func boolScore(hit bool, w float64) float64 {
	if !hit {
		return 0
	}
	return w
}

func concatStartupPacks(packs []vault.ContextPack) []vault.ContextPack {
	var out []vault.ContextPack
	for i := range packs {
		if packs[i].Startup && packs[i].Visibility != "secret" {
			out = append(out, packs[i])
		}
	}
	prio := map[string]int{"high": 3, "medium": 2, "low": 1, "": 0}
	sort.SliceStable(out, func(i, j int) bool { return prio[out[i].Priority] > prio[out[j].Priority] })
	return out
}

func contextTitle(c vault.ContextPack) string {
	if t := fmStr(c.FM, "title"); t != "" {
		return t
	}
	return c.Title
}

func findTypedByToolID[T any](items []T, want string, noteOf func(T) vault.Note) *T {
	want = strings.TrimSpace(want)
	for i := range items {
		n := noteOf(items[i])
		if vault.ToolID(n) == want || n.ID == vault.NormalizeID(want) {
			item := items[i]
			return &item
		}
	}
	return nil
}

func fmOr(fm map[string]interface{}, key, fallback string) string {
	if s := fmStr(fm, key); s != "" {
		return s
	}
	return fallback
}

func sectionText(body, heading string) string {
	sections := splitMarkdownSections(body)
	return strings.TrimSpace(sections[heading])
}

func linkObjects(r *depsHolder, links []vault.Link, body string, forward bool) []map[string]any {
	out := make([]map[string]any, 0, len(links))
	for _, l := range links {
		targetID := l.TargetID
		srcID := l.SourceID
		id := targetID
		ctxBody := body
		if !forward {
			id = srcID
			if n := r.d.Idx.NoteByID(srcID); n != nil {
				ctxBody = n.Body
			}
		}
		title := id
		if n := r.d.Idx.NoteByID(id); n != nil {
			title = n.Title
		}
		out = append(out, map[string]any{
			"id":        id,
			"title":     title,
			"path_hint": id,
			"context":   vault.LinkContext(ctxBody, l.Raw),
		})
	}
	return out
}

func backlinkEdges(r *depsHolder, id string) []vault.Link {
	var edges []vault.Link
	for _, src := range r.d.Idx.Backlinks(id) {
		raw := ""
		for _, l := range r.d.Idx.ForwardLinks(src) {
			if l.TargetID == id {
				raw = l.Raw
				break
			}
		}
		edges = append(edges, vault.Link{SourceID: src, TargetID: id, LinkType: "wikilink", Raw: raw})
	}
	return edges
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
