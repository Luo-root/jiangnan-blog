// Package auth 用 SQLite auth_tokens + 内存 cache 做 Bearer 鉴权。
//
// Token 不在 yaml。签发 / 轮换同步改 cache；reload 只给崩溃恢复。
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	StatusActive  = "active"
	StatusGrace   = "grace"
	StatusRevoked = "revoked"
)

// StandardScopes 是可签给 Agent 的 8 个 scope。admin:reindex 不在这里。
var StandardScopes = []string{
	"read:context",
	"read:knowledge",
	"read:project",
	"read:registry",
	"read:inbox",
	"write:proposal",
	"write:inbox",
	"ops:audit",
}

var standardSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(StandardScopes))
	for _, s := range StandardScopes {
		m[s] = struct{}{}
	}
	return m
}()

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("token not found")
	ErrNameTaken    = errors.New("active token name already exists")
	ErrInvalidScope = errors.New("invalid scope")
)

type ctxKey struct{}

type Token struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	TokenHash   string     `json:"-"`
	Scopes      []string   `json:"scopes"`
	Status      string     `json:"status"`
	GraceUntil  *time.Time `json:"-"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	UseCount    int        `json:"use_count"`
}

type AuthContext struct {
	ID         int64
	ClientID   string
	Scopes     []string
	Status     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	UseCount   int
}

func (a *AuthContext) HasScope(scope string) bool {
	if scope == "" {
		return true
	}
	for _, s := range a.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func WithAuth(ctx context.Context, a *AuthContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

func FromContext(ctx context.Context) (*AuthContext, bool) {
	a, ok := ctx.Value(ctxKey{}).(*AuthContext)
	return a, ok && a != nil
}

type Store struct {
	db         *sql.DB
	graceHours int
	mu         sync.RWMutex
	cache      map[string]*Token // token_hash -> row
	stop       chan struct{}
}

func Open(dbPath string, graceHours int) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{
		db:         db,
		graceHours: graceHours,
		cache:      map[string]*Token{},
		stop:       make(chan struct{}),
	}
	if err := s.reload(); err != nil {
		db.Close()
		return nil, err
	}
	go s.reloadLoop()
	return s, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS auth_tokens (
  id            INTEGER PRIMARY KEY,
  name          TEXT    NOT NULL,
  token_hash    TEXT    NOT NULL,
  scopes        TEXT    NOT NULL,
  status        TEXT    NOT NULL DEFAULT 'active',
  grace_until   TEXT,
  description   TEXT,
  created_at    TEXT    NOT NULL,
  created_by    TEXT    NOT NULL,
  last_used_at  TEXT,
  use_count     INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_tokens_active_name ON auth_tokens(name) WHERE status='active';
CREATE INDEX IF NOT EXISTS idx_auth_tokens_hash ON auth_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_status ON auth_tokens(status);
`

func (s *Store) Close() error {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	return s.db.Close()
}

func (s *Store) reloadLoop() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			_ = s.reload()
		}
	}
}

func (s *Store) reload() error {
	rows, err := s.db.Query(`SELECT id,name,token_hash,scopes,status,grace_until,description,created_at,created_by,last_used_at,use_count FROM auth_tokens WHERE status IN ('active','grace')`)
	if err != nil {
		return err
	}
	defer rows.Close()
	next := map[string]*Token{}
	for rows.Next() {
		tok, err := scanToken(rows)
		if err != nil {
			return err
		}
		next[tok.TokenHash] = tok
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.cache = next
	s.mu.Unlock()
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanToken(rs rowScanner) (*Token, error) {
	var (
		tok                  Token
		scopes               string
		grace, created, last sql.NullString
		desc, createdBy      sql.NullString
	)
	if err := rs.Scan(&tok.ID, &tok.Name, &tok.TokenHash, &scopes, &tok.Status, &grace, &desc, &created, &createdBy, &last, &tok.UseCount); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(scopes), &tok.Scopes); err != nil {
		return nil, err
	}
	if tok.Scopes == nil {
		tok.Scopes = []string{}
	}
	tok.Description = desc.String
	tok.CreatedBy = createdBy.String
	if created.Valid {
		t, err := time.Parse(time.RFC3339, created.String)
		if err != nil {
			return nil, err
		}
		tok.CreatedAt = t
	}
	if grace.Valid && grace.String != "" {
		t, err := time.Parse(time.RFC3339, grace.String)
		if err != nil {
			return nil, err
		}
		tok.GraceUntil = &t
	}
	if last.Valid && last.String != "" {
		t, err := time.Parse(time.RFC3339, last.String)
		if err != nil {
			return nil, err
		}
		tok.LastUsedAt = &t
	}
	return &tok, nil
}

func (s *Store) lookup(hash string) *Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t := s.cache[hash]
	if t == nil {
		return nil
	}
	cp := *t
	if t.GraceUntil != nil {
		g := *t.GraceUntil
		cp.GraceUntil = &g
	}
	if t.LastUsedAt != nil {
		l := *t.LastUsedAt
		cp.LastUsedAt = &l
	}
	cp.Scopes = append([]string(nil), t.Scopes...)
	return &cp
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func extractBearer(h http.Header) (string, bool) {
	hv := h.Get("Authorization")
	if !strings.HasPrefix(hv, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(hv, "Bearer "))
	if token == "" {
		return "", false
	}
	return token, true
}

func (s *Store) LookupHeader(h http.Header) (*AuthContext, error) {
	token, ok := extractBearer(h)
	if !ok {
		return nil, ErrUnauthorized
	}
	return s.lookupToken(token, false)
}

func (s *Store) AuthenticateHeader(h http.Header) (*AuthContext, error) {
	token, ok := extractBearer(h)
	if !ok {
		return nil, ErrUnauthorized
	}
	return s.lookupToken(token, true)
}

func (s *Store) lookupToken(plaintext string, touch bool) (*AuthContext, error) {
	row := s.lookup(HashToken(plaintext))
	if row == nil {
		return nil, ErrUnauthorized
	}
	switch row.Status {
	case StatusActive:
	case StatusGrace:
		if row.GraceUntil == nil || !time.Now().Before(*row.GraceUntil) {
			return nil, ErrUnauthorized
		}
	default:
		return nil, ErrUnauthorized
	}
	ac := &AuthContext{
		ID:         row.ID,
		ClientID:   row.Name,
		Scopes:     row.Scopes,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt,
		LastUsedAt: row.LastUsedAt,
		UseCount:   row.UseCount,
	}
	if touch {
		s.touch(row.ID, row.TokenHash)
	}
	return ac, nil
}

func (s *Store) touch(id int64, hash string) {
	now := time.Now().UTC()
	go func() {
		_, _ = s.db.Exec(`UPDATE auth_tokens SET last_used_at=?, use_count=use_count+1 WHERE id=?`, now.Format(time.RFC3339), id)
		s.mu.Lock()
		if t := s.cache[hash]; t != nil && t.ID == id {
			t.LastUsedAt = &now
			t.UseCount++
		}
		s.mu.Unlock()
	}()
}

func validateScopes(scopes []string) error {
	for _, sc := range scopes {
		if _, ok := standardSet[sc]; !ok {
			return fmt.Errorf("%w: %s", ErrInvalidScope, sc)
		}
	}
	return nil
}

func newPlaintext() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Store) Create(name, description, createdBy string, scopes []string) (plaintext string, tok *Token, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("name required")
	}
	if err := validateScopes(scopes); err != nil {
		return "", nil, err
	}
	if scopes == nil {
		scopes = []string{}
	}
	if createdBy == "" {
		createdBy = "admin"
	}
	plaintext, err = newPlaintext()
	if err != nil {
		return "", nil, err
	}
	hash := HashToken(plaintext)
	now := time.Now().UTC()
	scopeJSON, _ := json.Marshal(scopes)
	res, err := s.db.Exec(
		`INSERT INTO auth_tokens (name, token_hash, scopes, status, description, created_at, created_by, use_count) VALUES (?,?,?,?,?,?,?,0)`,
		name, hash, string(scopeJSON), StatusActive, description, now.Format(time.RFC3339), createdBy,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return "", nil, ErrNameTaken
		}
		return "", nil, err
	}
	id, _ := res.LastInsertId()
	tok = &Token{
		ID:          id,
		Name:        name,
		TokenHash:   hash,
		Scopes:      scopes,
		Status:      StatusActive,
		Description: description,
		CreatedAt:   now,
		CreatedBy:   createdBy,
	}
	s.mu.Lock()
	s.cache[hash] = tok
	s.mu.Unlock()
	return plaintext, tok, nil
}

func (s *Store) List() ([]Token, error) {
	rows, err := s.db.Query(`SELECT id,name,token_hash,scopes,status,grace_until,description,created_at,created_by,last_used_at,use_count FROM auth_tokens WHERE status != ? ORDER BY id DESC`, StatusRevoked)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Token
	for rows.Next() {
		tok, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tok.TokenHash = ""
		out = append(out, *tok)
	}
	return out, rows.Err()
}

func (s *Store) Rotate(id int64) (plaintext string, tok *Token, err error) {
	old, err := s.getByID(id)
	if err != nil {
		return "", nil, err
	}
	if old.Status != StatusActive {
		return "", nil, fmt.Errorf("only active tokens can be rotated")
	}
	plaintext, err = newPlaintext()
	if err != nil {
		return "", nil, err
	}
	newHash := HashToken(plaintext)
	now := time.Now().UTC()
	graceUntil := now.Add(time.Duration(s.graceHours) * time.Hour)
	scopeJSON, _ := json.Marshal(old.Scopes)

	tx, err := s.db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE auth_tokens SET status=?, grace_until=? WHERE id=?`, StatusGrace, graceUntil.Format(time.RFC3339), id); err != nil {
		return "", nil, err
	}
	res, err := tx.Exec(
		`INSERT INTO auth_tokens (name, token_hash, scopes, status, description, created_at, created_by, use_count) VALUES (?,?,?,?,?,?,?,0)`,
		old.Name, newHash, string(scopeJSON), StatusActive, old.Description, now.Format(time.RFC3339), old.CreatedBy,
	)
	if err != nil {
		return "", nil, err
	}
	newID, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return "", nil, err
	}

	s.mu.Lock()
	if s.graceHours <= 0 {
		delete(s.cache, old.TokenHash)
	} else {
		g := graceUntil
		old.Status = StatusGrace
		old.GraceUntil = &g
		s.cache[old.TokenHash] = old
	}
	tok = &Token{
		ID:          newID,
		Name:        old.Name,
		TokenHash:   newHash,
		Scopes:      append([]string(nil), old.Scopes...),
		Status:      StatusActive,
		Description: old.Description,
		CreatedAt:   now,
		CreatedBy:   old.CreatedBy,
	}
	s.cache[newHash] = tok
	s.mu.Unlock()
	return plaintext, tok, nil
}

func (s *Store) Revoke(id int64) error {
	old, err := s.getByID(id)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`UPDATE auth_tokens SET status=? WHERE id=?`, StatusRevoked, id); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.cache, old.TokenHash)
	s.mu.Unlock()
	return nil
}

func (s *Store) getByID(id int64) (*Token, error) {
	row := s.db.QueryRow(`SELECT id,name,token_hash,scopes,status,grace_until,description,created_at,created_by,last_used_at,use_count FROM auth_tokens WHERE id=?`, id)
	tok, err := scanToken(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return tok, err
}
