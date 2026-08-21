package auth

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func openStore(t *testing.T, grace int) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "auth.sqlite"), grace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndAuthenticate(t *testing.T) {
	s := openStore(t, 0)
	plain, tok, err := s.Create("minimax-code", "dev", "admin", []string{"read:context", "read:knowledge"})
	if err != nil {
		t.Fatal(err)
	}
	if tok.Status != StatusActive {
		t.Fatalf("status = %s", tok.Status)
	}
	h := http.Header{}
	h.Set("Authorization", "Bearer "+plain)
	ac, err := s.AuthenticateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if ac.ClientID != "minimax-code" {
		t.Fatalf("client = %s", ac.ClientID)
	}
	if !ac.HasScope("read:context") || ac.HasScope("write:proposal") {
		t.Fatalf("scopes = %v", ac.Scopes)
	}
}

func TestMissingTokenUnauthorized(t *testing.T) {
	s := openStore(t, 0)
	if _, err := s.AuthenticateHeader(http.Header{}); err != ErrUnauthorized {
		t.Fatalf("got %v", err)
	}
}

func TestInvalidScopeRejected(t *testing.T) {
	s := openStore(t, 0)
	_, _, err := s.Create("x", "", "admin", []string{"admin:reindex"})
	if err == nil {
		t.Fatal("expected invalid scope")
	}
}

func TestActiveNameUnique(t *testing.T) {
	s := openStore(t, 0)
	if _, _, err := s.Create("same", "", "admin", nil); err != nil {
		t.Fatal(err)
	}
	_, _, err := s.Create("same", "", "admin", nil)
	if err != ErrNameTaken {
		t.Fatalf("got %v", err)
	}
}

func TestRotateGraceZeroDropsOldCache(t *testing.T) {
	s := openStore(t, 0)
	oldPlain, tok, err := s.Create("bot", "", "admin", []string{"read:inbox"})
	if err != nil {
		t.Fatal(err)
	}
	newPlain, _, err := s.Rotate(tok.ID)
	if err != nil {
		t.Fatal(err)
	}
	hOld := http.Header{}
	hOld.Set("Authorization", "Bearer "+oldPlain)
	if _, err := s.AuthenticateHeader(hOld); err != ErrUnauthorized {
		t.Fatalf("old token should 401 immediately, got %v", err)
	}
	hNew := http.Header{}
	hNew.Set("Authorization", "Bearer "+newPlain)
	ac, err := s.AuthenticateHeader(hNew)
	if err != nil {
		t.Fatal(err)
	}
	if ac.ClientID != "bot" || ac.Status != StatusActive {
		t.Fatalf("new token auth = %+v", ac)
	}
	_, _, err = s.Create("bot", "after rotate", "admin", nil)
	if err != ErrNameTaken {
		t.Fatalf("active name should still be unique after rotate, got %v", err)
	}
}

func TestRotateGraceKeepsOldUntilDeadline(t *testing.T) {
	s := openStore(t, 24)
	oldPlain, tok, err := s.Create("bot", "", "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Rotate(tok.ID); err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Authorization", "Bearer "+oldPlain)
	ac, err := s.AuthenticateHeader(h)
	if err != nil {
		t.Fatalf("grace token should still work: %v", err)
	}
	if ac.Status != StatusGrace {
		t.Fatalf("status = %s", ac.Status)
	}
}

func TestRevokeDropsCache(t *testing.T) {
	s := openStore(t, 0)
	plain, tok, err := s.Create("bot", "", "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Revoke(tok.ID); err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Authorization", "Bearer "+plain)
	if _, err := s.AuthenticateHeader(h); err != ErrUnauthorized {
		t.Fatalf("revoked token still accepted: %v", err)
	}
}

func TestListHidesRevoked(t *testing.T) {
	s := openStore(t, 0)
	if _, _, err := s.Create("keep", "desc", "admin", nil); err != nil {
		t.Fatal(err)
	}
	_, tok, err := s.Create("drop", "", "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Revoke(tok.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "keep" || list[0].Description != "desc" {
		t.Fatalf("list = %+v", list)
	}
	for _, row := range list {
		if row.Status == StatusRevoked {
			t.Fatalf("revoked leaked: %+v", row)
		}
	}
}

func TestExpiredGraceRejected(t *testing.T) {
	s := openStore(t, 24)
	plain, tok, err := s.Create("bot", "", "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Rotate(tok.ID); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	for _, row := range s.cache {
		if row.Status == StatusGrace {
			past := time.Now().Add(-time.Minute)
			row.GraceUntil = &past
		}
	}
	s.mu.Unlock()
	h := http.Header{}
	h.Set("Authorization", "Bearer "+plain)
	if _, err := s.AuthenticateHeader(h); err != ErrUnauthorized {
		t.Fatalf("expired grace accepted: %v", err)
	}
}
