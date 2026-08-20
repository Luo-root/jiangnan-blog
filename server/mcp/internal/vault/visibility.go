package vault

import (
	"path/filepath"
	"strings"
)

var visibilityValues = map[string]struct{}{
	"public":  {},
	"private": {},
	"secret":  {},
	"draft":   {},
}

// defaultKeys 长前缀优先。运行时权威是 yaml；这里只是匹配顺序。
var defaultKeys = []string{
	"Workbase/context",
	"Workbase/skills",
	"Workbase/mcps",
	"文章",
	"项目",
	"友链",
	"部署溯源",
}

// ResolveVisibility 缺 visibility 时按一级目录缺省。secret 必须显式标注。
func ResolveVisibility(rel string, fmVis string, defaults map[string]string) string {
	if _, ok := visibilityValues[fmVis]; ok {
		return fmVis
	}
	rel = filepath.ToSlash(rel)
	for _, k := range defaultKeys {
		if rel == k || strings.HasPrefix(rel, k+"/") {
			if v := defaults[k]; v != "" {
				return v
			}
		}
	}
	if v := defaults["default"]; v != "" {
		return v
	}
	return "private"
}

// ClassifyKind 按路径写 kind。友链不入库。Workbase 下误放的 md 当 note。
func ClassifyKind(rel string) (kind string, skip bool) {
	rel = filepath.ToSlash(rel)
	switch {
	case rel == "友链" || strings.HasPrefix(rel, "友链/"):
		return "", true
	case strings.HasPrefix(rel, "文章/"):
		return "article", false
	case strings.HasPrefix(rel, "项目/"):
		return "project", false
	case strings.HasPrefix(rel, "Workbase/context/"):
		return "context_pack", false
	case strings.HasPrefix(rel, "Workbase/skills/"):
		return "skill", false
	case strings.HasPrefix(rel, "Workbase/mcps/"):
		return "mcp_server", false
	default:
		return "note", false
	}
}

// NormalizeID 查询 / 入库一律正斜杠。不自动补 .md。
func NormalizeID(id string) string {
	return filepath.ToSlash(id)
}

// NoteID 是 notes.id：vault 相对路径，正斜杠，含 .md。
func NoteID(vaultRoot, abs string) string {
	rel, err := filepath.Rel(vaultRoot, abs)
	if err != nil {
		rel = abs
	}
	return filepath.ToSlash(rel)
}

// ToolID 是对外工具 id：knowledge 用路径；project 用 slug；registry 用 frontmatter id。
func ToolID(n Note) string {
	switch n.Kind {
	case "project":
		return strings.TrimSuffix(filepath.Base(n.ID), ".md")
	case "skill", "mcp_server", "context_pack":
		if id, _ := n.FM["id"].(string); strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
		return strings.TrimSuffix(filepath.Base(n.ID), ".md")
	default:
		return n.ID
	}
}

func VisibleInList(vis string) bool {
	return vis == "public" || vis == "private" || vis == "draft"
}

func MatchScope(vis, scope string) bool {
	switch scope {
	case "all":
		return vis == "public" || vis == "private" || vis == "draft"
	case "public":
		return vis == "public"
	case "private":
		return vis == "private"
	default:
		return false
	}
}
