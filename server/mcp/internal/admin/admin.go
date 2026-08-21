// Package admin 提供 WebUI 后台 HTTP API（inbox 管理 + 热度数据 + 静态页面）。
package admin

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/apply"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/audit"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/auth"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/comment"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/inbox"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/index"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

//go:embed static
var staticFS embed.FS

// Handler 持有依赖并实现 http.Handler。
type Handler struct {
	Inbox            *inbox.Store
	Proposal         *proposal.Store
	Index            *index.Store
	Tokens           *auth.Store
	AdminUser        string
	AdminPassHash    string
	Sessions         *SessionStore
	Limiter          *LoginLimiter
	VaultRoot        string
	GitDir           string
	ExcludedSections []string
	VisDefault       map[string]string
	RebuildCmd       string
	HalfLifeDays     float64
	MinScore         float64
	Audit            *audit.Store
	SearchWeights    map[string]float64
	StartedAt        time.Time
	MCPListen        string
	AdminListen      string
	RuntimeDir       string
	IndexDB          string
	AuditDB          string
	AuthDB           string
	TemplatesDir     string
}

// ServeHTTP 路由分发。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS（开发期开放）。登录页是独立 session，不用浏览器弹窗 Basic Auth。
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")

	if strings.HasPrefix(path, "api/") && !isPublicAPI(path, r.Method) {
		if _, ok := h.sessionUser(r); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
	}

	switch {
	case path == "api/admin/login" && r.Method == http.MethodPost:
		h.login(w, r)
	case path == "api/admin/refresh" && r.Method == http.MethodPost:
		h.refresh(w, r)
	case path == "api/admin/logout" && r.Method == http.MethodPost:
		h.logout(w, r)
	case path == "api/admin/me" && r.Method == http.MethodGet:
		h.me(w, r)
	case path == "" || path == "index.html":
		h.serveStatic(w, r)
	case path == "api/inbox" && r.Method == http.MethodGet:
		h.listInbox(w, r)
	case path == "api/inbox" && r.Method == http.MethodPost:
		h.createInbox(w, r)
	case strings.HasPrefix(path, "api/inbox/") && r.Method == http.MethodGet:
		h.getInbox(w, r, strings.TrimPrefix(path, "api/inbox/"))
	case strings.HasPrefix(path, "api/inbox/") && r.Method == http.MethodPut:
		h.updateInbox(w, r, strings.TrimPrefix(path, "api/inbox/"))
	case path == "api/heat" && r.Method == http.MethodGet:
		h.getHeat(w, r)
	case path == "api/proposals" && r.Method == http.MethodGet:
		h.listProposals(w, r)
	case path == "api/proposals" && r.Method == http.MethodPost:
		h.createProposal(w, r)
	case strings.HasPrefix(path, "api/proposals/") && r.Method == http.MethodGet:
		h.getProposal(w, r, strings.TrimPrefix(path, "api/proposals/"))
	case strings.HasPrefix(path, "api/proposals/") && r.Method == http.MethodPut:
		h.reviewProposal(w, r, strings.TrimPrefix(path, "api/proposals/"))
	case strings.HasPrefix(path, "api/proposals/") && r.Method == http.MethodPatch:
		h.updateProposal(w, r, strings.TrimPrefix(path, "api/proposals/"))
	case path == "api/auth_tokens" && r.Method == http.MethodGet:
		h.listTokens(w, r)
	case path == "api/auth_tokens" && r.Method == http.MethodPost:
		h.createToken(w, r)
	case strings.HasSuffix(path, "/revoke") && strings.HasPrefix(path, "api/auth_tokens/") && r.Method == http.MethodPost:
		h.revokeToken(w, r, strings.TrimSuffix(strings.TrimPrefix(path, "api/auth_tokens/"), "/revoke"))
	case strings.HasSuffix(path, "/rotate") && strings.HasPrefix(path, "api/auth_tokens/") && r.Method == http.MethodPost:
		h.rotateToken(w, r, strings.TrimSuffix(strings.TrimPrefix(path, "api/auth_tokens/"), "/rotate"))
	case path == "api/audit/recent" && r.Method == http.MethodGet:
		h.listAudit(w, r)
	case path == "api/knowledge/search" && r.Method == http.MethodGet:
		h.searchKnowledge(w, r)
	case path == "api/knowledge" && r.Method == http.MethodGet:
		h.getKnowledge(w, r)
	case path == "api/system/health" && r.Method == http.MethodGet:
		h.systemHealth(w, r)
	case path == "api/git/history" && r.Method == http.MethodGet:
		h.gitHistory(w, r)
	case strings.HasPrefix(path, "api/git/diff/") && r.Method == http.MethodGet:
		h.gitDiff(w, r, strings.TrimPrefix(path, "api/git/diff/"))
	case path == "api/templates" && r.Method == http.MethodGet:
		h.listTemplates(w, r)
	case path == "api/templates" && r.Method == http.MethodPost:
		h.createTemplate(w, r)
	case strings.HasPrefix(path, "api/templates/") && r.Method == http.MethodPost:
		h.updateTemplate(w, r, strings.TrimPrefix(path, "api/templates/"))
	default:
		h.serveStatic(w, r)
	}
}

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	served := p
	b, err := fs.ReadFile(staticFS, "static/"+p)
	if err != nil {
		b, err = fs.ReadFile(staticFS, "static/index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		served = "index.html"
	}
	switch {
	case strings.HasSuffix(served, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(served, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(served, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(served, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(served, ".woff2"):
		w.Header().Set("Content-Type", "font/woff2")
	}
	w.Write(b)
}

// ---------------------------------------------------------------------------
// Inbox API
// ---------------------------------------------------------------------------

func (h *Handler) listInbox(w http.ResponseWriter, r *http.Request) {
	if h.Inbox == nil {
		writeJSON(w, http.StatusOK, []inbox.Summary{})
		return
	}
	items, err := h.Inbox.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if items == nil {
		items = []inbox.Summary{}
	}
	writeJSON(w, http.StatusOK, items)
}

type createReq struct {
	CreatedBy string   `json:"created_by"`
	Content   string   `json:"content"`
	Title     string   `json:"title"`
	Tags      []string `json:"tags"`
}

func (h *Handler) createInbox(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "webui"
	}
	id, err := h.Inbox.Append(req.CreatedBy, req.Content, req.Title, req.Tags)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "pending"})
}

func (h *Handler) getInbox(w http.ResponseWriter, r *http.Request, id string) {
	item, err := h.Inbox.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

type updateReq struct {
	Status  string         `json:"status"`
	Content string         `json:"content"`
	Title   string         `json:"title"`
	Tags    []string       `json:"tags"`
	Comment *comment.Input `json:"comment"`
}

func (h *Handler) updateInbox(w http.ResponseWriter, r *http.Request, id string) {
	var req updateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	user, _ := h.sessionUser(r)
	var cmt *comment.Comment
	if req.Comment != nil {
		c, err := comment.New("human", user, *req.Comment)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		cmt = &c
	}
	if err := h.Inbox.Update(id, inbox.Status(req.Status), req.Content, req.Title, req.Tags, cmt); err != nil {
		code := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(msg, "invalid status") {
			code = http.StatusConflict
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	item, err := h.Inbox.Get(id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": req.Status})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// ---------------------------------------------------------------------------
// Heat API
// ---------------------------------------------------------------------------

func (h *Handler) getHeat(w http.ResponseWriter, r *http.Request) {
	if h.Index == nil {
		writeJSON(w, http.StatusOK, []index.HotEntry{})
		return
	}
	hot := h.Index.Hot(h.HalfLifeDays, h.MinScore)
	if hot == nil {
		hot = []index.HotEntry{}
	}
	writeJSON(w, http.StatusOK, hot)
}

// ---------------------------------------------------------------------------
// Proposal API
// ---------------------------------------------------------------------------

func (h *Handler) listProposals(w http.ResponseWriter, r *http.Request) {
	if h.Proposal == nil {
		writeJSON(w, http.StatusOK, []proposal.Proposal{})
		return
	}
	props, err := h.Proposal.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if props == nil {
		props = []proposal.Proposal{}
	}
	writeJSON(w, http.StatusOK, props)
}

func (h *Handler) createProposal(w http.ResponseWriter, r *http.Request) {
	var p proposal.Proposal
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if p.CreatedBy == "" {
		p.CreatedBy = "webui"
	}
	created, err := h.Proposal.Create(p)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getProposal(w http.ResponseWriter, r *http.Request, id string) {
	p, err := h.Proposal.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type reviewReq struct {
	Status         string `json:"status"`
	KeepBaseCommit bool   `json:"keep_base_commit"`
}

// updateProposal 编辑 proposal 字段（pending / conflict 可编辑）。
// 设计文档 §21.5：支持「编辑后同意」和 conflict 救回。
type proposalUpdateHTTP struct {
	Reason     *string             `json:"reason,omitempty"`
	Target     *proposal.Target    `json:"target,omitempty"`
	Operation  *proposal.Operation `json:"operation,omitempty"`
	Payload    *proposal.Payload   `json:"payload,omitempty"`
	BaseCommit *string             `json:"base_commit,omitempty"`
	Comment    *comment.Input      `json:"comment,omitempty"`
}

func (h *Handler) updateProposal(w http.ResponseWriter, r *http.Request, id string) {
	var body proposalUpdateHTTP
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	patch := proposal.ProposalPatch{
		Reason:     body.Reason,
		Target:     body.Target,
		Operation:  body.Operation,
		Payload:    body.Payload,
		BaseCommit: body.BaseCommit,
	}
	if body.Comment != nil {
		user, _ := h.sessionUser(r)
		c, err := comment.New("human", user, *body.Comment)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		patch.Comment = &c
	}
	p, err := h.Proposal.Update(id, patch)
	if err != nil {
		code := proposalErrorCode(err)
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) reviewProposal(w http.ResponseWriter, r *http.Request, id string) {
	var req reviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	status := proposal.Status(req.Status)

	cur, err := h.Proposal.Get(id)
	if err != nil {
		writeJSON(w, proposalErrorCode(err), map[string]string{"error": err.Error()})
		return
	}
	if status == proposal.StatusApproved && cur.Status == proposal.StatusApplied {
		if cur.Receipt != nil {
			r := *cur.Receipt
			r.Replayed = true
			_ = h.Proposal.SetReceipt(id, &r)
			cur.Receipt = &r
		}
		writeJSON(w, http.StatusOK, cur)
		return
	}
	if status == proposal.StatusApproved && cur.Status == proposal.StatusConflict && !req.KeepBaseCommit {
		head := apply.Head(h.GitDir)
		if head != "" {
			base := head
			if _, err := h.Proposal.Update(id, proposal.ProposalPatch{BaseCommit: &base}); err != nil {
				log.Printf("refresh base_commit for %s: %v", id, err)
			}
		}
	}

	p, err := h.Proposal.UpdateStatus(id, status)
	if err != nil {
		code := proposalErrorCode(err)
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}

	if status == proposal.StatusApproved {
		if p.Receipt != nil && p.Receipt.Status == proposal.StatusApplied {
			r := *p.Receipt
			r.Replayed = true
			_ = h.Proposal.SetReceipt(id, &r)
			writeJSON(w, http.StatusOK, p)
			return
		}

		receipt, applyErr := apply.Apply(p, apply.Deps{
			VaultRoot:  h.VaultRoot,
			GitDir:     h.GitDir,
			VisDefault: h.VisDefault,
		})
		if applyErr != nil || receipt.Status == proposal.StatusConflict {
			if receipt == nil {
				receipt = &proposal.Receipt{Status: proposal.StatusConflict, AppliedAt: time.Now(), BaseCommit: p.BaseCommit}
			}
			log.Printf("apply proposal %s failed: %v", id, applyErr)
			_ = h.Proposal.SetReceipt(id, receipt)
			p, _ = h.Proposal.UpdateStatus(id, proposal.StatusConflict)
			writeJSON(w, http.StatusOK, p)
			return
		}

		_ = h.Proposal.SetReceipt(id, receipt)
		p, _ = h.Proposal.UpdateStatus(id, proposal.StatusApplied)

		if h.Index != nil {
			if err := h.Index.Rebuild(h.VaultRoot, h.ExcludedSections, h.VisDefault); err != nil {
				log.Printf("reindex after apply %s failed: %v", id, err)
			} else {
				log.Printf("reindex after apply %s: %d notes", id, len(h.Index.Notes()))
			}
		}
		if h.RebuildCmd != "" {
			cmd := exec.Command("sh", "-c", h.RebuildCmd)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Printf("rebuild after apply %s failed: %v — %s", id, err, out)
			} else {
				log.Printf("rebuild after apply %s ok", id)
			}
		}
		log.Printf("proposal %s → applied, sha256=%s, commit=%s", id, receipt.ContentSHA, receipt.Commit)
	}

	writeJSON(w, http.StatusOK, p)
}

// proposalErrorCode 将 proposal 层的错误映射为 HTTP 状态码。
func proposalErrorCode(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid review status"):
		return http.StatusBadRequest
	case strings.Contains(msg, "only pending") || strings.Contains(msg, "only pending or conflict") || strings.Contains(msg, "terminal or in-flight"):
		return http.StatusConflict
	case strings.Contains(msg, "only approved proposal"):
		return http.StatusConflict
	case strings.Contains(msg, "not found") || strings.Contains(msg, "no such file") || strings.Contains(msg, "missing frontmatter") || strings.Contains(msg, "unclosed frontmatter"):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

type createTokenReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
}

func (h *Handler) listTokens(w http.ResponseWriter, r *http.Request) {
	if h.Tokens == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token store not configured"})
		return
	}
	items, err := h.Tokens.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if items == nil {
		items = []auth.Token{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) createToken(w http.ResponseWriter, r *http.Request) {
	if h.Tokens == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token store not configured"})
		return
	}
	var req createTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	plaintext, tok, err := h.Tokens.Create(req.Name, req.Description, h.AdminUser, req.Scopes)
	if err != nil {
		code := http.StatusBadRequest
		if err == auth.ErrNameTaken {
			code = http.StatusConflict
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          tok.ID,
		"name":        tok.Name,
		"scopes":      tok.Scopes,
		"status":      tok.Status,
		"created_at":  tok.CreatedAt,
		"token":       plaintext,
		"description": tok.Description,
	})
}

func (h *Handler) rotateToken(w http.ResponseWriter, r *http.Request, idStr string) {
	if h.Tokens == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token store not configured"})
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	plaintext, tok, err := h.Tokens.Rotate(id)
	if err != nil {
		code := http.StatusBadRequest
		if err == auth.ErrNotFound {
			code = http.StatusNotFound
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     tok.ID,
		"name":   tok.Name,
		"scopes": tok.Scopes,
		"status": tok.Status,
		"token":  plaintext,
	})
}

func (h *Handler) revokeToken(w http.ResponseWriter, r *http.Request, idStr string) {
	if h.Tokens == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token store not configured"})
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.Tokens.Revoke(id); err != nil {
		code := http.StatusBadRequest
		if err == auth.ErrNotFound {
			code = http.StatusNotFound
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
