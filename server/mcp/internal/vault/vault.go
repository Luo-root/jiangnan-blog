// Package vault 提供 Obsidian Vault 内容读取能力。
package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Note 是 Vault 中一个 .md 文件的元数据摘要。
type Note struct {
	ID         string                 `json:"id"` // notes.id = vault 相对路径，正斜杠，含 .md
	Path       string                 `json:"path"`
	Title      string                 `json:"title"`
	Kind       string                 `json:"kind"`
	Section    string                 `json:"section"`
	Visibility string                 `json:"visibility"`
	Tags       []string               `json:"tags"`
	Summary    string                 `json:"summary"`
	Headings   []string               `json:"headings"`
	UpdatedAt  time.Time              `json:"updated_at"`
	FM         map[string]interface{} `json:"frontmatter"`
	Body       string                 `json:"-"`
}

// Index 是一次扫描的内存镜像。持久化走 SQLite，不写 JSON。
type Index struct {
	Notes     []Note
	Projects  []Project
	Skills    []Skill
	MCPS      []MCPServer
	Context   []ContextPack
	Links     []Link
	Backlinks map[string][]string
}

// Project 来自 项目/*.md。
type Project struct {
	Note
}

// Skill 来自 Workbase/skills/*.md。
type Skill struct {
	Note
	Risk   string      `json:"risk"`
	Source interface{} `json:"source"`
}

// MCPServer 来自 Workbase/mcps/*.md。
type MCPServer struct {
	Note
	Transport    string      `json:"transport"`
	EndpointHint string      `json:"endpoint_hint"`
	Auth         interface{} `json:"auth"`
	Risk         string      `json:"risk"`
}

// ContextPack 来自 Workbase/context/*.md。
type ContextPack struct {
	Note
	Startup  bool   `json:"startup"`
	Priority string `json:"priority"`
	Content  string `json:"content"`
}

// Link 是解析成功的 WikiLink 边。重名只记 raw，不建边。
type Link struct {
	SourceID string
	TargetID string
	LinkType string
	Raw      string
}

// Scan 扫描 vault 根目录。excluded 是一级目录名（.obsidian / .trash）。
func Scan(vaultRoot string, excluded []string, visDefault map[string]string) (*Index, error) {
	excl := map[string]bool{}
	for _, s := range excluded {
		excl[s] = true
	}
	if visDefault == nil {
		visDefault = map[string]string{}
	}

	idx := &Index{}
	entries, err := os.ReadDir(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("scan vault: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() || excl[e.Name()] {
			continue
		}
		section := e.Name()
		walkDir(vaultRoot, filepath.Join(vaultRoot, section), section, excl, visDefault, idx)
	}

	idx.Links, idx.Backlinks = buildLinks(idx.Notes)
	return idx, nil
}

func walkDir(vaultRoot, dir, section string, excl map[string]bool, visDefault map[string]string, idx *Index) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			if excl[e.Name()] {
				continue
			}
			walkDir(vaultRoot, filepath.Join(dir, e.Name()), section, excl, visDefault, idx)
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		abs := filepath.Join(dir, e.Name())
		id := NoteID(vaultRoot, abs)
		kind, skip := ClassifyKind(id)
		if skip {
			continue
		}
		note := parseNote(abs, section, id, kind, visDefault)
		if note == nil {
			continue
		}
		idx.Notes = append(idx.Notes, *note)

		switch kind {
		case "project":
			idx.Projects = append(idx.Projects, Project{Note: *note})
		case "skill":
			idx.Skills = append(idx.Skills, toSkill(*note))
		case "mcp_server":
			idx.MCPS = append(idx.MCPS, toMCPServer(*note))
		case "context_pack":
			cp := toContextPack(*note)
			cp.Content = note.Body
			if raw, err := os.ReadFile(abs); err == nil {
				cp.Content = string(raw)
			}
			idx.Context = append(idx.Context, cp)
		}
	}
}

func parseNote(abs, section, id, kind string, visDefault map[string]string) *Note {
	info, err := os.Stat(abs)
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	fm, body := splitFrontmatter(string(b))
	title := extractTitle(body, filepath.Base(abs))

	fmMap := map[string]interface{}{}
	if fm != "" {
		if err := yaml.Unmarshal([]byte(fm), &fmMap); err != nil {
			fmMap = map[string]interface{}{}
		}
	}

	fmVis, _ := fmMap["visibility"].(string)
	vis := ResolveVisibility(id, fmVis, visDefault)
	tags := extractTags(fmMap["tags"])
	summary := extractSummary(fmMap, body)
	updated := info.ModTime()
	if u := fmTime(fmMap["updated"]); !u.IsZero() {
		updated = u
	}

	return &Note{
		ID:         id,
		Path:       abs,
		Title:      title,
		Kind:       kind,
		Section:    section,
		Visibility: vis,
		Tags:       tags,
		Summary:    summary,
		Headings:   extractHeadings(body),
		UpdatedAt:  updated,
		FM:         fmMap,
		Body:       body,
	}
}

func extractTags(v interface{}) []string {
	var out []string
	switch t := v.(type) {
	case string:
		for _, s := range strings.Split(t, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case []interface{}:
		for _, item := range t {
			if s, ok := item.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func splitFrontmatter(text string) (fm string, body string) {
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

func extractTitle(body, filename string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return strings.TrimSuffix(filename, ".md")
}

func extractHeadings(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			continue
		}
		h := strings.TrimSpace(strings.TrimLeft(t, "#"))
		if h != "" {
			out = append(out, h)
		}
	}
	return out
}

func extractSummary(fm map[string]interface{}, body string) string {
	if s, _ := fm["summary"].(string); strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	var para []string
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(t, "#") {
			continue
		}
		para = append(para, t)
		if len(strings.Join(para, " ")) > 160 {
			break
		}
	}
	s := strings.Join(para, " ")
	if len([]rune(s)) > 180 {
		r := []rune(s)
		s = string(r[:180]) + "…"
	}
	return s
}

func fmTime(v interface{}) time.Time {
	s, _ := v.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func toSkill(n Note) Skill {
	r, _ := n.FM["risk"].(string)
	return Skill{Note: n, Risk: r, Source: n.FM["source"]}
}

func toMCPServer(n Note) MCPServer {
	t, _ := n.FM["transport"].(string)
	ep, _ := n.FM["endpoint"].(string)
	r, _ := n.FM["risk"].(string)
	return MCPServer{Note: n, Transport: t, EndpointHint: ep, Auth: n.FM["auth"], Risk: r}
}

func toContextPack(n Note) ContextPack {
	startup := false
	if b, ok := n.FM["startup"].(bool); ok {
		startup = b
	}
	p, _ := n.FM["priority"].(string)
	return ContextPack{Note: n, Startup: startup, Priority: p}
}

// ExtractWikiLinks 从 markdown 文本中提取 [[target]]。
func ExtractWikiLinks(text string) []string {
	var out []string
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
		if target != "" {
			out = append(out, target)
		}
		i += end + 2
	}
	return out
}

// ReadBody 读取 note 的正文（去 frontmatter）。
func ReadBody(abs string) string {
	b, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	_, body := splitFrontmatter(string(b))
	return body
}

func wikiKey(id string) string {
	return strings.ToLower(filepath.ToSlash(strings.TrimSuffix(id, ".md")))
}

func buildLinks(notes []Note) ([]Link, map[string][]string) {
	byPath := map[string]string{}
	byName := map[string][]string{}
	for i := range notes {
		id := notes[i].ID
		byPath[wikiKey(id)] = id
		name := strings.ToLower(strings.TrimSuffix(filepath.Base(id), ".md"))
		byName[name] = append(byName[name], id)
	}

	var links []Link
	bl := map[string][]string{}
	seen := map[string]struct{}{}
	for i := range notes {
		src := notes[i].ID
		for _, raw := range ExtractWikiLinks(notes[i].Body) {
			target := resolveWikiLink(raw, byPath, byName)
			if target == "" || target == src {
				continue
			}
			key := src + "\x00" + target
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			links = append(links, Link{SourceID: src, TargetID: target, LinkType: "wikilink", Raw: raw})
			bl[target] = append(bl[target], src)
		}
	}
	return links, bl
}

func resolveWikiLink(target string, byPath map[string]string, byName map[string][]string) string {
	key := wikiKey(target)
	if id, ok := byPath[key]; ok {
		return id
	}
	base := strings.ToLower(filepath.Base(key))
	ids := byName[base]
	if len(ids) == 1 {
		return ids[0]
	}
	return ""
}

// LinkContext 取链接前后约 50 字。
func LinkContext(body, raw string) string {
	idx := strings.Index(body, "[["+raw)
	if idx < 0 {
		idx = strings.Index(body, raw)
	}
	if idx < 0 {
		return ""
	}
	runes := []rune(body)
	start := 0
	pos := len([]rune(body[:idx]))
	if pos > 50 {
		start = pos - 50
	}
	end := pos + 50
	if end > len(runes) {
		end = len(runes)
	}
	return strings.TrimSpace(string(runes[start:end]))
}
