package apply

import (
	"errors"
	"strings"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

var (
	ErrSectionNotFound = errors.New("section_not_found")
	ErrUnsupportedOp   = errors.New("operation_not_supported")
)

func ApplyOp(text string, op proposal.Operation, payload string) (string, error) {
	return applyOp(text, op, payload)
}

func applyOp(text string, op proposal.Operation, payload string) (string, error) {
	payload = strings.TrimRight(payload, "\n") + "\n"
	switch op.Type {
	case "create_file", "register_item":
		return payload, nil
	case "append":
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return text + payload, nil
	case "append_section":
		return appendSectionText(text, op.Section, payload), nil
	case "patch_section":
		out, ok := patchSectionText(text, op.Section, payload)
		if !ok {
			return "", ErrSectionNotFound
		}
		return out, nil
	default:
		return "", ErrUnsupportedOp
	}
}

func appendSectionText(text, section, content string) string {
	start, end := findSection(text, section)
	if start < 0 {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		if text != "" && !strings.HasSuffix(text, "\n\n") {
			text += "\n"
		}
		return text + "## " + section + "\n\n" + content
	}
	inserted := text[:end]
	if !strings.HasSuffix(inserted, "\n") {
		inserted += "\n"
	}
	if !strings.HasSuffix(inserted, "\n\n") {
		inserted += "\n"
	}
	inserted += content
	if end < len(text) {
		if !strings.HasSuffix(inserted, "\n") {
			inserted += "\n"
		}
		inserted += text[end:]
	}
	return inserted
}

func patchSectionText(text, section, content string) (string, bool) {
	start, end := findSection(text, section)
	if start < 0 {
		return "", false
	}
	headingLine := ""
	for i := start; i < len(text) && text[i] != '\n'; i++ {
		headingLine += string(text[i])
	}
	patched := text[:start] + headingLine + "\n" + content
	if end < len(text) {
		if !strings.HasSuffix(patched, "\n") {
			patched += "\n"
		}
		patched += text[end:]
	}
	return patched, true
}

func findSection(text, name string) (start, end int) {
	normalize := func(s string) string {
		return strings.ToLower(strings.TrimSpace(s))
	}
	want := normalize(name)
	lines := strings.Split(text, "\n")
	offsets := make([]int, len(lines))
	off := 0
	for i, line := range lines {
		offsets[i] = off
		off += len(line) + 1
	}

	targetLevel := 0
	targetIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		level := 0
		for _, c := range trimmed {
			if c == '#' {
				level++
				continue
			}
			title := ""
			if c == ' ' && level < len(trimmed) {
				title = trimmed[level+1:]
			} else {
				title = trimmed[level:]
			}
			if normalize(title) == want {
				targetLevel = level
				targetIdx = i
			}
			break
		}
	}
	if targetIdx < 0 {
		return -1, -1
	}
	for i := targetIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		level := 0
		for _, c := range trimmed {
			if c == '#' {
				level++
			} else {
				break
			}
		}
		if level > 0 && level <= targetLevel {
			return offsets[targetIdx], offsets[i]
		}
	}
	return offsets[targetIdx], len(text)
}

func parseConflicts(text string) []proposal.ConflictRegion {
	var out []proposal.ConflictRegion
	for {
		i := strings.Index(text, "<<<<<<<")
		if i < 0 {
			break
		}
		j := strings.Index(text[i:], ">>>>>>>")
		excerpt := text[i:]
		if j >= 0 {
			excerpt = text[i : i+j+7]
			text = text[i+j+7:]
		} else {
			text = ""
		}
		if len(excerpt) > 400 {
			excerpt = excerpt[:400]
		}
		out = append(out, proposal.ConflictRegion{Excerpt: excerpt})
		if j < 0 {
			break
		}
	}
	return out
}

func frontmatterConflict(text string) bool {
	t := strings.TrimLeft(text, "\n")
	if !strings.HasPrefix(t, "---") {
		return false
	}
	rest := strings.TrimPrefix(t, "---")
	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return strings.Contains(t, "<<<<<<<")
	}
	return strings.Contains(rest[:idx], "<<<<<<<")
}
