// Package admin 提供 WebUI 后台 HTTP API（inbox 管理 + 热度数据 + 静态页面）。
package admin

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/apply"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/auth"
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
	VaultRoot        string
	GitDir           string
	ExcludedSections []string
	VisDefault       map[string]string
	RebuildCmd       string
}

// ServeHTTP 路由分发。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// admin 认证（独立于 MCP Bearer token，§21.5）
	// OPTIONS 预检请求跳过认证（浏览器不会在预检时带 Authorization）
	if r.Method != http.MethodOptions && h.AdminUser != "" && !h.checkAdminAuth(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Workbase Admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// CORS（开发期开放）
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")

	switch {
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
	default:
		h.serveStatic(w, r)
	}
}

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	b, err := fs.ReadFile(staticFS, "static/"+p)
	if err != nil {
		// SPA fallback
		b, err = fs.ReadFile(staticFS, "static/index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	if strings.HasSuffix(p, ".html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else if strings.HasSuffix(p, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	} else if strings.HasSuffix(p, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Write(b)
}

// ---------------------------------------------------------------------------
// Inbox API
// ---------------------------------------------------------------------------

func (h *Handler) listInbox(w http.ResponseWriter, r *http.Request) {
	items, err := h.Inbox.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type createReq struct {
	CreatedBy string `json:"created_by"`
	Content   string `json:"content"`
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
	id, err := h.Inbox.Append(req.CreatedBy, req.Content, "", nil)
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
	Status  string `json:"status"`
	Content string `json:"content"`
}

func (h *Handler) updateInbox(w http.ResponseWriter, r *http.Request, id string) {
	var req updateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.Inbox.Update(id, inbox.Status(req.Status), req.Content, "", nil); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": req.Status})
}

// ---------------------------------------------------------------------------
// Heat API
// ---------------------------------------------------------------------------

func (h *Handler) getHeat(w http.ResponseWriter, r *http.Request) {
	hot := h.Index.Hot()
	writeJSON(w, http.StatusOK, hot)
}

// ---------------------------------------------------------------------------
// Proposal API
// ---------------------------------------------------------------------------

func (h *Handler) listProposals(w http.ResponseWriter, r *http.Request) {
	props, err := h.Proposal.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
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
func (h *Handler) updateProposal(w http.ResponseWriter, r *http.Request, id string) {
	var patch proposal.ProposalPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
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
	case strings.Contains(msg, "only pending") || strings.Contains(msg, "only pending or conflict"):
		return http.StatusConflict
	case strings.Contains(msg, "only approved proposal"):
		return http.StatusConflict
	case strings.Contains(msg, "no such file") || strings.Contains(msg, "missing frontmatter") || strings.Contains(msg, "unclosed frontmatter"):
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

// checkAdminAuth 验证 HTTP Basic Auth。
func (h *Handler) checkAdminAuth(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok || user != h.AdminUser {
		return false
	}
	sum := sha256.Sum256([]byte(pass))
	hash := hex.EncodeToString(sum[:])
	return hash == h.AdminPassHash
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
