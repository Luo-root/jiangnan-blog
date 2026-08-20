package apply

import (
	"os"
	"strings"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
)

func Preview(p *proposal.Proposal, vaultRoot string) string {
	abs, rel, err := ResolvePath(vaultRoot, p.Target.Path)
	if err != nil {
		return "target_path_invalid"
	}
	before := ""
	if b, err := os.ReadFile(abs); err == nil {
		before = string(b)
	}
	after, err := applyOp(before, p.Operation, p.Payload.Content)
	if err != nil {
		return err.Error()
	}
	var b strings.Builder
	b.WriteString("--- a/")
	b.WriteString(rel)
	b.WriteString("\n+++ b/")
	b.WriteString(rel)
	b.WriteString("\n")
	if before == after {
		b.WriteString("(no change)\n")
		return b.String()
	}
	b.WriteString(simpleDiff(before, after))
	return b.String()
}

func simpleDiff(before, after string) string {
	bl := strings.Split(strings.TrimSuffix(before, "\n"), "\n")
	al := strings.Split(strings.TrimSuffix(after, "\n"), "\n")
	if before == "" {
		bl = nil
	}
	if after == "" {
		al = nil
	}
	n, m := len(bl), len(al)
	i := 0
	for i < n && i < m && bl[i] == al[i] {
		i++
	}
	j := 0
	for i+j < n && i+j < m && bl[n-1-j] == al[m-1-j] {
		j++
	}
	var b strings.Builder
	for _, line := range bl[i : n-j] {
		b.WriteString("-")
		b.WriteString(line)
		b.WriteString("\n")
	}
	for _, line := range al[i : m-j] {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
