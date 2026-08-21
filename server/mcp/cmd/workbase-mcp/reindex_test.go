package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/config"
	"github.com/Luo-root/jiangnan-blog-agent-workbase/mcp/internal/index"
)

func TestInternalReindexIgnoresRemoteAddr(t *testing.T) {
	dir := t.TempDir()
	idx, err := index.Open(filepath.Join(dir, "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	cfg := &config.Config{}
	cfg.Vault.Root = dir

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/reindex", handleInternalReindex(cfg, idx))

	req := httptest.NewRequest(http.MethodPost, "/internal/reindex", nil)
	req.RemoteAddr = "203.0.113.9:54321"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("body = %s", body)
	}
}

func TestInternalReindexRejectsPut(t *testing.T) {
	dir := t.TempDir()
	idx, err := index.Open(filepath.Join(dir, "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	cfg := &config.Config{}
	cfg.Vault.Root = dir
	h := handleInternalReindex(cfg, idx)

	req := httptest.NewRequest(http.MethodPut, "/internal/reindex", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}
