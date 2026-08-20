package audit

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendListMinFieldsAndNoRaw(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "audit.sqlite"), 90, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	raw := `{"query":"kubernetes","token":"sk-live-should-not-appear"}`
	s.Append(Entry{
		TS:           time.Now(),
		Tool:         "knowledge.search",
		ClientID:     "bot",
		Scopes:       []string{"read:knowledge", "read:context"},
		ArgsDigest:   Digest(raw),
		ResultStatus: "success",
		DurationMS:   12,
		TargetPath:   "文章/hello.md",
	})
	got := s.List(Filter{Limit: 10})
	if len(got) != 1 {
		t.Fatalf("got %d", len(got))
	}
	e := got[0]
	if e.Tool != "knowledge.search" || e.ClientID != "bot" || e.ResultStatus != "success" {
		t.Fatalf("%+v", e)
	}
	if e.ArgsDigest == "" || e.ArgsDigest == raw {
		t.Fatal("args_digest must be hash, not raw")
	}
	if e.DurationMS != 12 {
		t.Fatalf("duration = %d", e.DurationMS)
	}
	if len(e.Scopes) != 2 {
		t.Fatalf("scopes = %v", e.Scopes)
	}
	if strings.Contains(e.ArgsDigest, "sk-live") || strings.Contains(e.Error, "sk-live") {
		t.Fatalf("token leaked: %+v", e)
	}
}

func TestListFilters(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "audit.sqlite"), 90, 50)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.Append(Entry{Tool: "knowledge.get", ClientID: "a", ResultStatus: "success", DurationMS: 1})
	s.Append(Entry{Tool: "proposal.create", ClientID: "b", ResultStatus: "error", DurationMS: 2, Error: "stale_base"})
	s.Append(Entry{Tool: "knowledge.get", ClientID: "a", ResultStatus: "forbidden", DurationMS: 3})

	if n := s.List(Filter{Tool: "knowledge.get"}); len(n) != 2 {
		t.Fatalf("tool filter = %d", len(n))
	}
	if n := s.List(Filter{ClientID: "b"}); len(n) != 1 || n[0].ResultStatus != "error" {
		t.Fatalf("client filter = %+v", n)
	}
	if n := s.List(Filter{ResultStatus: "forbidden"}); len(n) != 1 {
		t.Fatalf("status filter = %d", len(n))
	}
}

func TestRetentionDeletesOld(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "audit.sqlite"), 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.Append(Entry{TS: time.Now().Add(-48 * time.Hour), Tool: "old", ClientID: "x", ResultStatus: "success"})
	s.Append(Entry{TS: time.Now(), Tool: "new", ClientID: "x", ResultStatus: "success"})
	got := s.List(Filter{Limit: 10})
	if len(got) != 1 || got[0].Tool != "new" {
		t.Fatalf("retention = %+v", got)
	}
}
