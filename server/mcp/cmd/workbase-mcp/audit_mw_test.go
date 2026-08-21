package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/audit"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/auth"
)

func openAuditAuth(t *testing.T) (*auth.Store, *audit.Store, string) {
	t.Helper()
	tokens, err := auth.Open(filepath.Join(t.TempDir(), "auth.sqlite"), 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tokens.Close() })
	ad, err := audit.Open(filepath.Join(t.TempDir(), "audit.sqlite"), 90, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ad.Close() })
	plain, _, err := tokens.Create("bot", "", "admin", []string{"read:context"})
	if err != nil {
		t.Fatal(err)
	}
	return tokens, ad, plain
}

func callMW(t *testing.T, mw func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error), ctx context.Context, name string, args map[string]any, header http.Header) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{Header: header, Params: mcp.CallToolParams{Name: name, Arguments: args}}
	res, err := mw(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAuthAuditMiddlewareRecordsAfterNext(t *testing.T) {
	tokens, ad, plain := openAuditAuth(t)
	ok := authAuditMiddleware(tokens, ad)(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultStructuredOnly(map[string]any{"ok": true}), nil
	})
	fail := authAuditMiddleware(tokens, ad)(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("note not found"), nil
	})

	res := callMW(t, ok, context.Background(), "workbase.identity", map[string]any{"token": "sk-live-should-not-appear"}, nil)
	if !res.IsError || !strings.Contains(toolErrorText(res), "unauthorized") {
		t.Fatalf("want unauthorized, got %+v", res)
	}

	h := http.Header{}
	h.Set("Authorization", "Bearer "+plain)
	res = callMW(t, ok, context.Background(), "context.startup", map[string]any{"id": "Workbase/context/profile.md"}, h)
	if res.IsError {
		t.Fatalf("startup: %s", toolErrorText(res))
	}

	res = callMW(t, ok, context.Background(), "proposal.create", map[string]any{
		"target": map[string]any{"path": "文章/hello.md"},
	}, h)
	if !res.IsError || !strings.Contains(toolErrorText(res), "forbidden") {
		t.Fatalf("want forbidden, got %s", toolErrorText(res))
	}

	res = callMW(t, fail, context.Background(), "context.get", map[string]any{"id": "missing"}, h)
	if !res.IsError {
		t.Fatal("want tool error")
	}

	got := ad.List(audit.Filter{Limit: 20})
	if len(got) != 4 {
		t.Fatalf("entries = %d %+v", len(got), got)
	}
	byStatus := map[string]audit.Entry{}
	for _, e := range got {
		byStatus[e.ResultStatus] = e
		if strings.Contains(e.ArgsDigest, "sk-live") || strings.Contains(e.Error, plain) || strings.Contains(e.ClientID, plain) {
			t.Fatalf("secret leaked: %+v", e)
		}
	}
	if byStatus["unauthorized"].Tool != "workbase.identity" || byStatus["unauthorized"].ClientID != "" {
		t.Fatalf("unauthorized = %+v", byStatus["unauthorized"])
	}
	if byStatus["success"].Tool != "context.startup" || byStatus["success"].ClientID != "bot" {
		t.Fatalf("success = %+v", byStatus["success"])
	}
	if len(byStatus["success"].Scopes) == 0 {
		t.Fatal("success scopes empty")
	}
	if byStatus["forbidden"].Tool != "proposal.create" || byStatus["forbidden"].TargetPath != "文章/hello.md" {
		t.Fatalf("forbidden = %+v", byStatus["forbidden"])
	}
	if byStatus["error"].Tool != "context.get" || !strings.Contains(byStatus["error"].Error, "note not found") {
		t.Fatalf("error = %+v", byStatus["error"])
	}
}

func TestBearerHTTPAuthUnauthorized(t *testing.T) {
	tokens, err := auth.Open(filepath.Join(t.TempDir(), "auth.sqlite"), 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tokens.Close() })
	innerRan := false
	h := bearerHTTPAuth(tokens, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerRan = true
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if innerRan {
		t.Fatal("inner ran")
	}
}
