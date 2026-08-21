package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/apply"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/audit"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/search"
)

func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	if h.Audit == nil {
		writeJSON(w, http.StatusOK, []audit.Entry{})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	f := audit.Filter{
		Limit:        limit,
		Tool:         r.URL.Query().Get("tool"),
		ClientID:     r.URL.Query().Get("client_id"),
		ResultStatus: r.URL.Query().Get("result_status"),
	}
	if since := r.URL.Query().Get("since"); since != "" {
		t, err := parseRFC3339(since)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_argument: since"})
			return
		}
		f.Since = t
	}
	if until := r.URL.Query().Get("until"); until != "" {
		t, err := parseRFC3339(until)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_argument: until"})
			return
		}
		f.Until = t
	}
	if !f.Since.IsZero() && !f.Until.IsZero() && f.Until.Before(f.Since) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_argument"})
		return
	}
	items := h.Audit.List(f)
	if items == nil {
		items = []audit.Entry{}
	}
	writeJSON(w, http.StatusOK, items)
}

func parseRFC3339(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func (h *Handler) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kind := r.URL.Query().Get("kind")
	visibility := r.URL.Query().Get("visibility")
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if q == "" && kind == "" && visibility == "" && tag == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"results": []any{},
			"message": "输入关键词或选一个过滤条件",
		})
		return
	}
	if h.Index == nil {
		writeJSON(w, http.StatusOK, search.EmptyResult(q))
		return
	}
	sortMode := r.URL.Query().Get("sort")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	limit = search.ClipLimit(limit, search.DefaultLimit, search.MaxLimit)

	weights := search.DefaultWeights()
	if len(h.SearchWeights) > 0 {
		weights = search.MergeWeights(h.SearchWeights)
	}
	tokens := search.Tokenize(q)
	listOnly := q == ""
	now := time.Now()
	started := now
	var hits []search.Result
	for _, n := range h.Index.Notes() {
		if kind != "" && n.Kind != kind {
			continue
		}
		if visibility != "" && n.Visibility != visibility {
			continue
		}
		if !search.HasTag(n.Tags, tag) {
			continue
		}
		doc := search.FromNote(n, h.Index.Count(n.ID), h.Index.LastAccess(n.ID), len(h.Index.Backlinks(n.ID)))
		if listOnly {
			hits = append(hits, search.Result{
				ID:         doc.ID,
				Title:      doc.Title,
				PathHint:   doc.ID,
				Kind:       doc.Kind,
				Visibility: doc.Visibility,
				Summary:    doc.Summary,
				Score:      0,
				Signals: map[string]float64{
					"access":  search.Access(doc.AccessCount, doc.LastAccess, now, h.HalfLifeDays, 1),
					"recency": search.Recency(doc.UpdatedAt, now, 1),
				},
			})
			continue
		}
		hit, ok := search.Score(doc, tokens, weights, nil, h.HalfLifeDays, now)
		if !ok {
			continue
		}
		hits = append(hits, hit)
	}
	search.SortHits(hits, sortMode)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	if len(hits) == 0 {
		out := search.EmptyResult(q)
		out["elapsed_ms"] = time.Since(started).Milliseconds()
		writeJSON(w, http.StatusOK, out)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"results":    hits,
		"elapsed_ms": time.Since(started).Milliseconds(),
		"query_echo": q,
	})
}

func (h *Handler) getKnowledge(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	if h.Index == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	n := h.Index.NoteByID(id)
	if n == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          n.ID,
		"title":       n.Title,
		"kind":        n.Kind,
		"visibility":  n.Visibility,
		"summary":     n.Summary,
		"tags":        n.Tags,
		"headings":    n.Headings,
		"updated_at":  n.UpdatedAt.Format(time.RFC3339),
		"frontmatter": n.FM,
		"body":        n.Body,
		"path_hint":   n.ID,
	})
}

func (h *Handler) systemHealth(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	started := h.StartedAt
	if started.IsZero() {
		started = now
	}
	notes, projects, skills, mcps, context := 0, 0, 0, 0, 0
	if h.Index != nil {
		notes, projects, skills, mcps, context = h.Index.Stats()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"now":        now.Format(time.RFC3339),
		"uptime_sec": int(now.Sub(started).Seconds()),
		"listen": map[string]string{
			"mcp":   h.MCPListen,
			"admin": h.AdminListen,
		},
		"paths": map[string]string{
			"vault_root": h.VaultRoot,
			"git_dir":    h.GitDir,
			"runtime":    h.RuntimeDir,
			"index_db":   h.IndexDB,
			"audit_db":   h.AuditDB,
			"auth_db":    h.AuthDB,
		},
		"sqlite": map[string]any{
			"index_bytes": fileSize(h.IndexDB),
			"audit_bytes": fileSize(h.AuditDB),
			"auth_bytes":  fileSize(h.AuthDB),
		},
		"index": map[string]int{
			"notes":    notes,
			"projects": projects,
			"skills":   skills,
			"mcps":     mcps,
			"context":  context,
		},
		"disk":     diskUsage(h.RuntimeDir),
		"git_head": apply.Head(h.GitDir),
	})
}

func (h *Handler) gitHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	commits, err := apply.Log(h.GitDir, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if commits == nil {
		commits = []apply.LogEntry{}
	}
	writeJSON(w, http.StatusOK, commits)
}

func (h *Handler) gitDiff(w http.ResponseWriter, r *http.Request, commit string) {
	commit = strings.TrimSpace(commit)
	if !validCommitSHA(commit) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid commit"})
		return
	}
	patch, err := apply.Patch(h.GitDir, commit)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"commit": commit, "diff": patch})
}

type Template struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Reason      string   `json:"reason"`
	TargetType  string   `json:"target_type"`
	Operation   string   `json:"operation"`
	Section     string   `json:"section,omitempty"`
	Payload     string   `json:"payload"`
	Title       string   `json:"title,omitempty"`
	Content     string   `json:"content,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	UpdatedAt   string   `json:"updated_at"`
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	items, err := h.loadTemplates()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var t Template
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	t.Kind = normalizeTemplateKind(t.Kind)
	if t.ID == "" {
		t.ID = slugID(t.Name)
	}
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if t.Scopes == nil {
		t.Scopes = []string{}
	}
	if err := h.saveTemplate(t); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	cur, err := h.loadTemplate(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var patch Template
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if patch.Kind != "" {
		cur.Kind = normalizeTemplateKind(patch.Kind)
	}
	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Title != "" {
		cur.Title = patch.Title
	}
	if patch.Content != "" {
		cur.Content = patch.Content
	}
	if patch.Tags != nil {
		cur.Tags = patch.Tags
	}
	if patch.Description != "" {
		cur.Description = patch.Description
	}
	if patch.Reason != "" {
		cur.Reason = patch.Reason
	}
	if patch.TargetType != "" {
		cur.TargetType = patch.TargetType
	}
	if patch.Operation != "" {
		cur.Operation = patch.Operation
	}
	cur.Section = patch.Section
	if patch.Payload != "" {
		cur.Payload = patch.Payload
	}
	if patch.Scopes != nil {
		cur.Scopes = patch.Scopes
	}
	cur.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := h.saveTemplate(cur); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cur)
}

func (h *Handler) templatesDir() string {
	if h.TemplatesDir != "" {
		return h.TemplatesDir
	}
	if h.RuntimeDir != "" {
		return filepath.Join(h.RuntimeDir, "templates")
	}
	return ""
}

func (h *Handler) loadTemplates() ([]Template, error) {
	dir := h.templatesDir()
	if dir == "" {
		return []Template{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Template{}, nil
		}
		return nil, err
	}
	out := make([]Template, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t, err := h.loadTemplate(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (h *Handler) loadTemplate(id string) (Template, error) {
	var t Template
	dir := h.templatesDir()
	b, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if t.ID == "" {
		t.ID = id
	}
	t.Kind = normalizeTemplateKind(t.Kind)
	return t, nil
}

func normalizeTemplateKind(kind string) string {
	switch kind {
	case "inbox", "proposal", "token":
		return kind
	default:
		return "proposal"
	}
}

func (h *Handler) saveTemplate(t Template) error {
	dir := h.templatesDir()
	if dir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, t.ID+".json"), b, 0644)
}

func validCommitSHA(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func slugID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '_' || r == '-':
			return '-'
		default:
			return -1
		}
	}, s)
	s = strings.Trim(s, "-")
	if s == "" {
		s = "tpl-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	return s
}

func fileSize(path string) int64 {
	if path == "" {
		return 0
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

func diskUsage(path string) map[string]any {
	out := map[string]any{"path": path, "free_bytes": int64(0), "total_bytes": int64(0)}
	if path == "" {
		return out
	}
	free, total, err := diskFree(path)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	out["free_bytes"] = free
	out["total_bytes"] = total
	return out
}
