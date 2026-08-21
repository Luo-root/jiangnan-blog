package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/audit"
)

func TestAuditListRecentSchemaFields(t *testing.T) {
	ad, err := audit.Open(filepath.Join(t.TempDir(), "audit.sqlite"), 90, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ad.Close() })
	raw := `{"query":"kubernetes","token":"sk-live-should-not-appear"}`
	ad.Append(audit.Entry{
		TS:           time.Now().Add(-time.Minute),
		Tool:         "knowledge.search",
		ClientID:     "bot",
		Scopes:       []string{"read:knowledge"},
		ArgsDigest:   audit.Digest(raw),
		ResultStatus: "success",
		DurationMS:   9,
		TargetPath:   "文章/hello.md",
	})
	ad.Append(audit.Entry{
		Tool:         "proposal.create",
		ClientID:     "other",
		Scopes:       []string{"write:proposal"},
		ArgsDigest:   audit.Digest(`{"target":{"path":"x"}}`),
		ResultStatus: "error",
		DurationMS:   3,
		Error:        "stale_base",
	})

	r := &depsHolder{d: Deps{Audit: ad}}
	m := wrap(r.handleAuditListRecent(context.Background(), callReq(map[string]any{
		"tool":          "knowledge.search",
		"client_id":     "bot",
		"result_status": "success",
		"limit":         10,
	}))).mapOK(t)
	entries := asMaps(m["entries"])
	if len(entries) != 1 {
		t.Fatalf("filtered = %+v", entries)
	}
	e := entries[0]
	for _, k := range []string{"ts", "tool", "client_id", "scopes", "args_digest", "result_status", "duration_ms"} {
		if e[k] == nil || e[k] == "" {
			t.Fatalf("missing %s: %+v", k, e)
		}
	}
	digest, _ := e["args_digest"].(string)
	if strings.Contains(digest, "sk-live") || strings.Contains(digest, "kubernetes") {
		t.Fatalf("raw args leaked: %s", digest)
	}
	if e["target_path"] != "文章/hello.md" {
		t.Fatalf("target = %v", e["target_path"])
	}

	msg := wrap(r.handleAuditListRecent(context.Background(), callReq(map[string]any{"result_status": "nope"}))).errText(t)
	if !strings.Contains(msg, "invalid_argument") {
		t.Fatalf("status err = %q", msg)
	}
	msg = wrap(r.handleAuditListRecent(context.Background(), callReq(map[string]any{"since": "not-a-time"}))).errText(t)
	if !strings.Contains(msg, "invalid_argument") {
		t.Fatalf("since err = %q", msg)
	}
}
