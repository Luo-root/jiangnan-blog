// Package apply 负责执行 Proposal 批准后的实际文件写入（§17.3 apply 流程）。
//
// 支持 operation.type：create_file / append / append_section / patch_section /
// replace_frontmatter / register_item。
//
// 返回 Receipt 用于幂等校验和审计。
package apply

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

// Deps 是 apply 所需的外部依赖。
type Deps struct {
	VaultRoot string // 内容事实源根目录
}

// Apply 执行 proposal 的写入操作，返回 receipt。
// 幂等：如果 proposal 已有 receipt 且 status=applied，返回原 receipt 并标记 replayed=true。
func Apply(p *proposal.Proposal, deps Deps) (*proposal.Receipt, error) {
	// 幂等检查
	if p.Receipt != nil && p.Receipt.Status == proposal.StatusApplied {
		r := *p.Receipt
		r.Replayed = true
		return &r, nil
	}

	absPath := filepath.Join(deps.VaultRoot, filepath.FromSlash(p.Target.Path))

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("create target dir: %w", err)
	}

	// 执行写入
	switch p.Operation.Type {
	case "create_file", "register_item":
		if err := createFile(absPath, p.Payload.Content); err != nil {
			return nil, err
		}
	case "append":
		if err := appendFile(absPath, p.Payload.Content); err != nil {
			return nil, err
		}
	case "append_section":
		if err := appendSection(absPath, p.Operation.Section, p.Payload.Content); err != nil {
			return nil, err
		}
	case "patch_section":
		if err := patchSection(absPath, p.Operation.Section, p.Payload.Content); err != nil {
			return nil, err
		}
	case "replace_frontmatter":
		if err := replaceFrontmatter(absPath, p.Payload.Content); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported operation: %s", p.Operation.Type)
	}

	// 生成 receipt
	now := time.Now()
	receipt := &proposal.Receipt{
		Status:    proposal.StatusApplied,
		AppliedAt: now,
		Replayed:  false,
	}

	// 计算目标文件 SHA-256
	b, err := os.ReadFile(absPath)
	if err == nil {
		h := sha256.Sum256(b)
		receipt.ContentSHA = fmt.Sprintf("%x", h)
	}

	return receipt, nil
}

// createFile 新建文件（目标已存在则报错）。
func createFile(absPath, content string) error {
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("target file already exists: %s", absPath)
	}
	return os.WriteFile(absPath, []byte(content+"\n"), 0644)
}

// appendFile 在文件末尾追加内容。
func appendFile(absPath, content string) error {
	existing, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	text := string(existing)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += content + "\n"
	return os.WriteFile(absPath, []byte(text), 0644)
}

// appendSection 在指定标题下追加内容（标题不存在则追加到文件末尾）。
func appendSection(absPath, section, content string) error {
	existing, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	text := string(existing)

	start, end := findSection(text, section)
	if start < 0 {
		// 标题不存在，追加到末尾
		if !strings.HasSuffix(text, "\n\n") {
			text = strings.TrimRight(text, "\n") + "\n\n"
		}
		text += content + "\n"
		return os.WriteFile(absPath, []byte(text), 0644)
	}

	// 在 section 结尾（下一同级或更高级标题之前）插入
	insert := end
	inserted := text[:insert]
	if !strings.HasSuffix(inserted, "\n\n") {
		inserted = strings.TrimRight(inserted, "\n") + "\n"
	}
	inserted += content + "\n"
	if end < len(text) {
		inserted += "\n" + text[end:]
	}
	return os.WriteFile(absPath, []byte(inserted), 0644)
}

// patchSection 替换指定标题下的全部内容。
func patchSection(absPath, section, content string) error {
	existing, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	text := string(existing)

	start, end := findSection(text, section)
	if start < 0 {
		return fmt.Errorf("section %q not found in %s", section, absPath)
	}

	// 保留标题行，替换标题之后到下一同级标题之间的内容
	headingLine := ""
	for i := start; i < len(text) && text[i] != '\n'; i++ {
		headingLine += string(text[i])
	}

	patched := text[:start]
	patched += headingLine + "\n" + content + "\n"
	if end < len(text) {
		patched += "\n" + text[end:]
	}
	return os.WriteFile(absPath, []byte(patched), 0644)
}

// replaceFrontmatter 替换文件的 YAML frontmatter。
func replaceFrontmatter(absPath, newFM string) error {
	existing, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	text := string(existing)

	// 找到原有 frontmatter 边界
	t := strings.TrimLeft(text, "\n")
	if !strings.HasPrefix(t, "---") {
		// 没有 frontmatter，在前面插入
		text = "---\n" + newFM + "\n---\n\n" + text
		return os.WriteFile(absPath, []byte(text), 0644)
	}

	rest := strings.TrimPrefix(t, "---\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fmt.Errorf("unclosed frontmatter in %s", absPath)
	}
	// 替换 frontmatter
	body := rest[idx+4:] // 跳过 "\n---"
	text = "---\n" + newFM + "\n---\n" + body
	return os.WriteFile(absPath, []byte(text), 0644)
}

// findSection 在 markdown 文本中查找指定标题，返回该 section 的起始位置和结束位置（字节偏移）。
// 匹配规则：大小写不敏感，trim 比较。起始位置在标题行首，结束位置在下一个同级或更高级标题之前。
// 未找到返回 (-1, -1)。
func findSection(text, name string) (start, end int) {
	normalize := func(s string) string {
		return strings.ToLower(strings.TrimSpace(s))
	}
	want := normalize(name)

	lines := strings.Split(text, "\n")

	// 计算每行的字节偏移
	offsets := make([]int, len(lines))
	off := 0
	for i, line := range lines {
		offsets[i] = off
		off += len(line) + 1 // +1 for \n
	}

	// 第一遍：找目标标题
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
			} else {
				if c == ' ' && normalize(trimmed[level+1:]) == want {
					targetLevel = level
					targetIdx = i
				}
				break
			}
		}
	}
	if targetIdx < 0 {
		return -1, -1
	}

	// 第二遍：找结束位置（下一个同级或更高级标题）
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
		if level <= targetLevel {
			return offsets[targetIdx], offsets[i]
		}
	}
	return offsets[targetIdx], len(text)
}
