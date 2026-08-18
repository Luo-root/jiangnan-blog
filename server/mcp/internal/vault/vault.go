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
	ID         string                 `json:"id"` // vault 相对路径（去 .md，正斜杠）
	Path       string                 `json:"path"`
	Title      string                 `json:"title"`
	Section    string                 `json:"section"` // 一级目录（栏目）
	Visibility string                 `json:"visibility"`
	Tags       []string               `json:"tags"`
	UpdatedAt  time.Time              `json:"updated_at"`
	FM         map[string]interface{} `json:"frontmatter"`
}

// Index 是所有索引数据的容器。
type Index struct {
	Notes     []Note              `json:"notes"`
	Projects  []Project           `json:"projects"`
	Skills    []Skill             `json:"skills"`
	MCPS      []MCPServer         `json:"mcps"`
	Context   []ContextPack       `json:"context"`
	Backlinks map[string][]string `json:"-"` // id → 指向它的 ids（运行时计算，不持久化到 JSON）
}

// Project 来自 项目/*.md frontmatter。
type Project struct {
	Note
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
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

// Scan 扫描 vault 根目录，返回索引。
func Scan(vaultRoot string, excludedSections []string) (*Index, error) {
	excl := map[string]bool{}
	for _, s := range excludedSections {
		excl[s] = true
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
		walkDir(vaultRoot, filepath.Join(vaultRoot, section), section, excl, idx)
	}

	// 构建反向链接索引（跑完所有 note 后）
	idx.Backlinks = buildBacklinks(vaultRoot, idx.Notes)

	return idx, nil
}

func walkDir(vaultRoot, dir, section string, excl map[string]bool, idx *Index) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			if excl[e.Name()] {
				continue
			}
			walkDir(vaultRoot, filepath.Join(dir, e.Name()), section, excl, idx)
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		abs := filepath.Join(dir, e.Name())
		note := parseNote(abs, section)
		if note == nil {
			continue
		}
		note.ID = noteID(vaultRoot, abs)
		idx.Notes = append(idx.Notes, *note)

		switch section {
		case "项目":
			idx.Projects = append(idx.Projects, toProject(*note))
		case "Workbase":
			kind, _ := note.FM["kind"].(string)
			switch kind {
			case "skill":
				idx.Skills = append(idx.Skills, toSkill(*note))
			case "mcp_server":
				idx.MCPS = append(idx.MCPS, toMCPServer(*note))
			}
			if note.FM["type"] != nil && fmt.Sprint(note.FM["type"]) == "context_pack" {
				cp := toContextPack(*note)
				b, _ := os.ReadFile(abs)
				cp.Content = string(b)
				idx.Context = append(idx.Context, cp)
			}
		}
	}
}

func noteID(vaultRoot, abs string) string {
	rel, err := filepath.Rel(vaultRoot, abs)
	if err != nil {
		rel = abs
	}
	rel = filepath.ToSlash(strings.TrimSuffix(rel, ".md"))
	return rel
}

func parseNote(abs, section string) *Note {
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
		// frontmatter 用 yaml.v3 解析，正确支持 tags 列表、source/auth 嵌套 map。
		if err := yaml.Unmarshal([]byte(fm), &fmMap); err != nil {
			fmMap = map[string]interface{}{}
		}
	}

	vis, _ := fmMap["visibility"].(string)
	if vis == "" {
		vis = "public" // 缺省兼容
	}

	tags := extractTags(fmMap["tags"])

	return &Note{
		Path:       abs,
		Title:      title,
		Section:    section,
		Visibility: vis,
		Tags:       tags,
		UpdatedAt:  info.ModTime(),
		FM:         fmMap,
	}
}

// extractTags 兼容 tags 的两种写法：逗号分隔字符串 / YAML 列表。
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
			return strings.TrimLeft(t, "# ")
		}
	}
	return strings.TrimSuffix(filename, ".md")
}

func toProject(n Note) Project {
	n2, _ := n.FM["name"].(string)
	if n2 == "" {
		n2 = n.Title
	}
	s, _ := n.FM["summary"].(string)
	st, _ := n.FM["status"].(string)
	return Project{Note: n, Name: n2, Summary: s, Status: st}
}

func toSkill(n Note) Skill {
	r, _ := n.FM["risk"].(string)
	src := n.FM["source"]
	return Skill{Note: n, Risk: r, Source: src}
}

func toMCPServer(n Note) MCPServer {
	t, _ := n.FM["transport"].(string)
	ep, _ := n.FM["endpoint"].(string)
	r, _ := n.FM["risk"].(string)
	auth := n.FM["auth"]
	return MCPServer{Note: n, Transport: t, EndpointHint: ep, Auth: auth, Risk: r}
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

// buildBacklinks 为每个 note 构建反向链接索引。
// 返回 map[id]→[]ids（指向该 id 的 note id 列表）。
func buildBacklinks(vaultRoot string, notes []Note) map[string][]string {
	bl := map[string][]string{}
	// 先建立 filename→id 索引加速解析
	nameIdx := map[string]string{} // 文件名（去.md）→ id
	for i := range notes {
		name := strings.TrimSuffix(filepath.Base(notes[i].Path), ".md")
		// 多级路径也要匹配：文章/运维学习笔记/01-Kubernetes/Ingress-Nginx-Max
		nameIdx[strings.ToLower(name)] = notes[i].ID
		nameIdx[strings.ToLower(notes[i].ID)] = notes[i].ID
	}

	for i := range notes {
		body := ReadBody(notes[i].Path)
		links := ExtractWikiLinks(body)
		for _, target := range links {
			candidate := resolveWikiLink(vaultRoot, target, nameIdx)
			if candidate != "" && candidate != notes[i].ID {
				bl[candidate] = append(bl[candidate], notes[i].ID)
			}
		}
	}
	return bl
}

// resolveWikiLink 将 WikiLink target 解析为 note id（vault 相对路径）。
func resolveWikiLink(vaultRoot string, target string, nameIdx map[string]string) string {
	target = filepath.ToSlash(strings.TrimSuffix(target, ".md"))
	// 1. 直接路径匹配
	if id, ok := nameIdx[strings.ToLower(target)]; ok {
		return id
	}
	// 2. 文件名匹配
	base := strings.ToLower(filepath.Base(target))
	if id, ok := nameIdx[base]; ok {
		return id
	}
	// 3. 直接文件存在
	abs := filepath.Join(vaultRoot, filepath.FromSlash(target)+".md")
	if _, err := os.Stat(abs); err == nil {
		return target
	}
	return ""
}
