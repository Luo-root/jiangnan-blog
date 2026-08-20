package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeYAML(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func validYAML() string {
	return `
server:
  listen: 127.0.0.1:8787
admin:
  listen: 127.0.0.1:8788
vault:
  root: D:/Data/工作台
workbase:
  root: D:/Data/工作台/Workbase
  runtime: .workbase
admin_auth:
  user: admin
  pass_hash: abcdef
schema:
  visibility_policy:
    public:  "可公开展示与索引"
    private: "授权 Agent 可读"
    secret:  "默认不暴露给远程 MCP"
    draft:   "草稿"
`
}

func TestLoadRequiredFields(t *testing.T) {
	dir := t.TempDir()
	p := writeYAML(t, dir, "config.yaml", validYAML())
	c, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.AdminAuth.User != "admin" {
		t.Fatalf("user = %q", c.AdminAuth.User)
	}
	if c.Schema.VisibilityPolicy["public"] == "" {
		t.Fatal("visibility_policy.public empty")
	}
	if c.Inbox.RetentionDays != DefaultInboxRetentionDays {
		t.Fatalf("inbox retention fallback = %d", c.Inbox.RetentionDays)
	}
	if c.Auth.GracePeriodHours != 0 {
		t.Fatalf("grace default = %d", c.Auth.GracePeriodHours)
	}
	if len(c.Schema.SensitivePatterns) != 0 {
		t.Fatalf("sensitive_patterns should default empty, got %v", c.Schema.SensitivePatterns)
	}
	if !filepath.IsAbs(c.Workbase.Runtime) {
		t.Fatalf("runtime not abs: %s", c.Workbase.Runtime)
	}
	if !strings.HasSuffix(c.AuthDB(), filepath.Join("auth.sqlite")) && !strings.Contains(c.AuthDB(), "auth.sqlite") {
		t.Fatalf("auth db path = %s", c.AuthDB())
	}
}

func TestLoadMissingVisibilityPolicy(t *testing.T) {
	dir := t.TempDir()
	body := strings.Replace(validYAML(), "    public:  \"可公开展示与索引\"\n", "", 1)
	p := writeYAML(t, dir, "bad.yaml", body)
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), "visibility_policy") {
		t.Fatalf("want visibility_policy error, got %v", err)
	}
}

func TestLoadMissingAdminAuth(t *testing.T) {
	dir := t.TempDir()
	body := strings.Replace(validYAML(), "  user: admin\n  pass_hash: abcdef\n", "  user: \"\"\n  pass_hash: \"\"\n", 1)
	p := writeYAML(t, dir, "bad.yaml", body)
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), "admin_auth") {
		t.Fatalf("want admin_auth error, got %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeYAML(t, dir, "bad.yaml", "admin_auth: [\n")
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
