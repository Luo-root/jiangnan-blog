package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/apply"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/auth"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/comment"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/inbox"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/proposal"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/sanitize"
)

type proposalCreateArgs struct {
	ExpectedBase string `json:"expected_base"`
	Target       struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Path string `json:"path"`
	} `json:"target"`
	Operation struct {
		Type    string `json:"type"`
		Section string `json:"section"`
	} `json:"operation"`
	Payload struct {
		Format  string `json:"format"`
		Content string `json:"content"`
	} `json:"payload"`
	Reason string `json:"reason"`
	Risk   struct {
		Level   string   `json:"level"`
		Reasons []string `json:"reasons"`
	} `json:"risk"`
}

func (r *depsHolder) handleProposalCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args proposalCreateArgs
	if err := req.BindArguments(&args); err != nil {
		return mcp.NewToolResultError("required field missing"), nil
	}
	if args.Target.Type == "" || args.Operation.Type == "" {
		return mcp.NewToolResultError("required field missing"), nil
	}
	if args.Target.Path == "" {
		return mcp.NewToolResultError("required field missing"), nil
	}
	if args.Payload.Content == "" {
		return mcp.NewToolResultError("required field missing"), nil
	}
	if args.Payload.Format == "" {
		args.Payload.Format = "markdown"
	}
	if args.Payload.Format != "markdown" {
		return mcp.NewToolResultError("invalid_argument: payload.format"), nil
	}
	if (args.Operation.Type == "append_section" || args.Operation.Type == "patch_section") && args.Operation.Section == "" {
		return mcp.NewToolResultError("required field missing"), nil
	}
	if !apply.Allowed(args.Target.Type, args.Operation.Type) {
		return mcp.NewToolResultError("operation_not_supported"), nil
	}
	abs, rel, err := apply.ResolvePath(r.d.VaultRoot, args.Target.Path)
	if err != nil {
		return mcp.NewToolResultError("target_path_invalid"), nil
	}
	if !apply.FenceClosed(args.Payload.Content) {
		return mcp.NewToolResultError("invalid_markdown_fence"), nil
	}

	head := baseCommit(r.d.GitDir)
	if args.ExpectedBase != "" && head != "" && args.ExpectedBase != head {
		return mcp.NewToolResultError("stale_base: 你读的已经不是最新，当前 HEAD=" + head), nil
	}
	base := head
	if args.ExpectedBase != "" {
		base = args.ExpectedBase
	}

	create := args.Operation.Type == "create_file" || args.Operation.Type == "register_item"
	_, statErr := os.Stat(abs)
	exists := statErr == nil
	if create && exists {
		return mcp.NewToolResultError("target_already_exists"), nil
	}
	if !create && !exists {
		return mcp.NewToolResultError("target_not_found"), nil
	}
	if args.Operation.Type == "patch_section" && exists {
		b, _ := os.ReadFile(abs)
		if _, err := apply.ApplyOp(string(b), proposal.Operation{Type: args.Operation.Type, Section: args.Operation.Section}, args.Payload.Content); err != nil {
			return mcp.NewToolResultError("section_not_found"), nil
		}
	}

	visDefault := map[string]string{}
	patterns := []string{}
	if r.d.Cfg != nil {
		visDefault = r.d.Cfg.Schema.VisibilityDefault
		patterns = r.d.Cfg.Schema.SensitivePatterns
	}
	if exists && apply.FileVisibility(abs, rel, visDefault) == "secret" {
		return mcp.NewToolResultError("visibility_not_writable"), nil
	}

	checks := []string{"target_path", "operation_matrix", "markdown_fence"}
	if create {
		checks = append(checks, "target_already_exists")
	} else {
		checks = append(checks, "target_not_found")
	}
	warnings := sanitize.Find(patterns, args.Payload.Content, args.Reason)
	if len(patterns) > 0 {
		checks = append(checks, "sensitive_patterns")
	}

	p := proposal.Proposal{
		Kind:       args.Target.Type,
		Reason:     args.Reason,
		CreatedBy:  clientID(ctx),
		BaseCommit: base,
		Target: proposal.Target{
			Type: args.Target.Type,
			ID:   args.Target.ID,
			Path: rel,
		},
		Operation: proposal.Operation{
			Type:    args.Operation.Type,
			Section: args.Operation.Section,
		},
		Payload: proposal.Payload{
			Format:  "markdown",
			Content: args.Payload.Content,
		},
		Risk: proposal.Risk{
			Level:   args.Risk.Level,
			Reasons: args.Risk.Reasons,
		},
		Validation: proposal.Validation{
			OK:       true,
			Checks:   checks,
			Warnings: warnings,
		},
	}
	created, err := r.d.Proposal.Create(p)
	if err != nil {
		return errResult("create proposal", err), nil
	}
	return jsonResult(map[string]any{
		"id":          created.ID,
		"status":      created.Status,
		"base_commit": created.BaseCommit,
		"created_at":  created.CreatedAt.Format(time.RFC3339),
		"created_by":  created.CreatedBy,
		"diff":        apply.Preview(created, r.d.VaultRoot),
		"validation":  created.Validation,
		"comments":    comment.Slice(created.Comments),
	}), nil
}

func (r *depsHolder) handleProposalList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	props, err := r.d.Proposal.List()
	if err != nil {
		return errResult("list proposals", err), nil
	}
	status := req.GetString("status", "all")
	createdBy := req.GetString("created_by", "")
	since := req.GetString("since", "")
	limit := req.GetInt("limit", 50)
	if limit <= 0 {
		limit = 50
	}
	var sinceT time.Time
	if since != "" {
		sinceT, _ = time.Parse(time.RFC3339, since)
	}
	out := make([]map[string]any, 0)
	for _, p := range props {
		if status != "" && status != "all" && string(p.Status) != status {
			continue
		}
		if createdBy != "" && p.CreatedBy != createdBy {
			continue
		}
		if !sinceT.IsZero() && p.CreatedAt.Before(sinceT) {
			continue
		}
		out = append(out, map[string]any{
			"id":          p.ID,
			"kind":        p.Kind,
			"status":      p.Status,
			"target_path": p.Target.Path,
			"created_at":  p.CreatedAt.Format(time.RFC3339),
			"created_by":  p.CreatedBy,
			"reason":      p.Reason,
			"risk_level":  p.Risk.Level,
		})
		if len(out) >= limit {
			break
		}
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
		return mcp.NewToolResultError(err.Error()), nil
	}
	return r.proposalGetResult(p), nil
}

type proposalUpdateArgs struct {
	ID        string `json:"id"`
	Reason    string `json:"reason"`
	Target    *proposal.Target
	Operation *proposal.Operation
	Payload   *proposal.Payload
	Comment   *comment.Input `json:"comment"`
}

func (r *depsHolder) handleProposalUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args proposalUpdateArgs
	if err := req.BindArguments(&args); err != nil && req.GetString("id", "") == "" {
		return mcp.NewToolResultError("required field missing"), nil
	}
	id := strings.TrimSpace(args.ID)
	if id == "" {
		id = req.GetString("id", "")
	}
	if id == "" {
		return mcp.NewToolResultError("required field missing"), nil
	}
	raw, _ := req.GetArguments()["target"]
	if t := objectAs[proposal.Target](raw); t != nil {
		args.Target = t
	}
	raw, _ = req.GetArguments()["operation"]
	if op := objectAs[proposal.Operation](raw); op != nil {
		args.Operation = op
	}
	raw, _ = req.GetArguments()["payload"]
	if p := objectAs[proposal.Payload](raw); p != nil {
		args.Payload = p
	}

	hasField := args.Reason != "" || args.Target != nil || args.Operation != nil || args.Payload != nil || args.Comment != nil
	if !hasField {
		return mcp.NewToolResultError("required field missing"), nil
	}

	var cmt *comment.Comment
	if args.Comment != nil {
		c, err := comment.New("agent", clientID(ctx), *args.Comment)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		cmt = &c
	}
	if args.Payload != nil {
		if args.Payload.Format == "" {
			args.Payload.Format = "markdown"
		}
		if args.Payload.Format != "markdown" {
			return mcp.NewToolResultError("invalid_argument: payload.format"), nil
		}
		if !apply.FenceClosed(args.Payload.Content) {
			return mcp.NewToolResultError("invalid_markdown_fence"), nil
		}
	}

	patch := proposal.ProposalPatch{
		Target:    args.Target,
		Operation: args.Operation,
		Payload:   args.Payload,
		Comment:   cmt,
	}
	if args.Reason != "" {
		reason := args.Reason
		patch.Reason = &reason
	}
	p, err := r.d.Proposal.Update(id, patch)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return r.proposalGetResult(p), nil
}

func (r *depsHolder) proposalGetResult(p *proposal.Proposal) *mcp.CallToolResult {
	result := map[string]any{
		"id":          p.ID,
		"kind":        p.Kind,
		"status":      p.Status,
		"base_commit": p.BaseCommit,
		"created": map[string]any{
			"by":     p.CreatedBy,
			"at":     p.CreatedAt.Format(time.RFC3339),
			"reason": p.Reason,
		},
		"target":     p.Target,
		"operation":  p.Operation,
		"payload":    p.Payload,
		"validation": p.Validation,
		"diff":       apply.Preview(p, r.d.VaultRoot),
		"comments":   comment.Slice(p.Comments),
	}
	if p.Risk.Level != "" || len(p.Risk.Reasons) > 0 {
		result["risk"] = p.Risk
	}
	if p.Receipt != nil {
		result["receipt"] = p.Receipt
	}
	return jsonResult(result)
}

func objectAs[T any](raw any) *T {
	if raw == nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return &out
}

func (r *depsHolder) handleInboxAppend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("required field content missing"), nil
	}
	title := req.GetString("title", "")
	id, err := r.d.Inbox.Append(clientID(ctx), content, title, req.GetStringSlice("tags", nil))
	if err != nil {
		return errResult("inbox append", err), nil
	}
	item, err := r.d.Inbox.Get(id)
	if err != nil {
		return errResult("inbox get", err), nil
	}
	return jsonResult(inboxItemOut(r, item)), nil
}

func (r *depsHolder) handleInboxUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("required argument id missing"), nil
	}
	args := req.GetArguments()
	status := inbox.Status(req.GetString("status", ""))
	content := req.GetString("content", "")
	title := req.GetString("title", "")
	var tags []string
	if raw, ok := args["tags"]; ok {
		tags = stringSlice(raw)
	}
	cmtIn := objectAs[comment.Input](args["comment"])
	if status == "" && content == "" && title == "" && !okKey(args, "tags") && cmtIn == nil {
		return mcp.NewToolResultError("required field missing"), nil
	}
	var cmt *comment.Comment
	if cmtIn != nil {
		c, err := comment.New("agent", clientID(ctx), *cmtIn)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		cmt = &c
	}
	if err := r.d.Inbox.Update(id, status, content, title, tags, cmt); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") || os.IsNotExist(err) {
			return mcp.NewToolResultError("inbox not found: " + id), nil
		}
		return mcp.NewToolResultError(msg), nil
	}
	item, err := r.d.Inbox.Get(id)
	if err != nil {
		return errResult("inbox get", err), nil
	}
	return jsonResult(inboxItemOut(r, item)), nil
}

func okKey(args map[string]any, key string) bool {
	_, ok := args[key]
	return ok
}

func stringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (r *depsHolder) handleInboxList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	items, err := r.d.Inbox.List()
	if err != nil {
		return errResult("inbox list", err), nil
	}
	status := req.GetString("status", "all")
	createdBy := req.GetString("created_by", "")
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if status != "" && status != "all" && string(it.Status) != status {
			continue
		}
		if createdBy != "" && it.CreatedBy != createdBy {
			continue
		}
		out = append(out, map[string]any{
			"id":            it.ID,
			"created_at":    it.CreatedAt.Format(time.RFC3339),
			"updated_at":    it.UpdatedAt.Format(time.RFC3339),
			"created_by":    it.CreatedBy,
			"title":         it.Title,
			"description":   it.Description,
			"summary":       it.Summary,
			"status":        it.Status,
			"comment_count": it.CommentCount,
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
		return mcp.NewToolResultError("inbox not found: " + id), nil
	}
	return jsonResult(inboxItemOut(r, item)), nil
}

func inboxItemOut(r *depsHolder, item *inbox.Item) map[string]any {
	out := map[string]any{
		"id":         item.ID,
		"status":     item.Status,
		"content":    item.Content,
		"created_at": item.CreatedAt.Format(time.RFC3339),
		"created_by": item.CreatedBy,
		"updated_at": item.UpdatedAt.Format(time.RFC3339),
		"title":      item.Title,
		"tags":       item.Tags,
		"comments":   comment.Slice(item.Comments),
		"created": map[string]any{
			"by": item.CreatedBy,
			"at": item.CreatedAt.Format(time.RFC3339),
		},
	}
	if ws := inboxWarnings(r, item.Content); len(ws) > 0 {
		out["warnings"] = ws
	}
	return out
}

func inboxWarnings(r *depsHolder, content string) []string {
	if r.d.Cfg == nil {
		return nil
	}
	return sanitize.Find(r.d.Cfg.Schema.SensitivePatterns, content)
}

func clientID(ctx context.Context) string {
	if ac, ok := auth.FromContext(ctx); ok && ac.ClientID != "" {
		return ac.ClientID
	}
	return "unknown"
}
