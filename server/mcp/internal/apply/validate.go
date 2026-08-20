package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/vault"
	"gopkg.in/yaml.v3"
)

var allowedOps = map[string]map[string]bool{
	"note":         {"append": true, "append_section": true},
	"context_pack": {"append_section": true, "patch_section": true},
	"project":      {"patch_section": true, "append_section": true},
	"article":      {"create_file": true},
	"skill":        {"register_item": true},
	"mcp_server":   {"register_item": true},
}

func Allowed(targetType, opType string) bool {
	ops := allowedOps[targetType]
	return ops[opType]
}

func ResolvePath(vaultRoot, rel string) (abs, slash string, err error) {
	slash = filepath.ToSlash(strings.TrimSpace(rel))
	slash = strings.TrimPrefix(slash, "/")
	if slash == "" || strings.Contains(slash, "..") {
		return "", "", fmt.Errorf("target_path_invalid")
	}
	abs = filepath.Clean(filepath.Join(vaultRoot, filepath.FromSlash(slash)))
	root := filepath.Clean(vaultRoot)
	relToRoot, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(relToRoot, "..") {
		return "", "", fmt.Errorf("target_path_invalid")
	}
	return abs, filepath.ToSlash(relToRoot), nil
}

func FenceClosed(md string) bool {
	n := 0
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			n++
		}
	}
	return n%2 == 0
}

func FileVisibility(abs, rel string, defaults map[string]string) string {
	b, err := os.ReadFile(abs)
	if err != nil {
		return vault.ResolveVisibility(rel, "", defaults)
	}
	return ContentVisibility(string(b), rel, defaults)
}

func ContentVisibility(text, rel string, defaults map[string]string) string {
	fmVis := ""
	t := strings.TrimLeft(text, "\n")
	if strings.HasPrefix(t, "---") {
		rest := strings.TrimPrefix(t, "---")
		if strings.HasPrefix(rest, "\n") {
			rest = rest[1:]
		}
		idx := strings.Index(rest, "\n---")
		if idx >= 0 {
			var fm map[string]any
			if err := yaml.Unmarshal([]byte(rest[:idx]), &fm); err == nil {
				if s, ok := fm["visibility"].(string); ok {
					fmVis = s
				}
			}
		}
	}
	return vault.ResolveVisibility(rel, fmVis, defaults)
}
