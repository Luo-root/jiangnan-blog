// Package audit 把工具调用记进 SQLite（SCHEMA §20 / 设计 §20.3）。
//
// 最小字段集：ts / tool / client_id / scopes / args_digest / result_status / duration_ms。
// 不存 token 原文、token hash、args 原文、secret 正文。
package audit

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Entry struct {
	TS           time.Time `json:"ts"`
	Tool         string    `json:"tool"`
	ClientID     string    `json:"client_id"`
	Scopes       []string  `json:"scopes"`
	ArgsDigest   string    `json:"args_digest"`
	ResultStatus string    `json:"result_status"`
	DurationMS   int       `json:"duration_ms"`
	Error        string    `json:"error,omitempty"`
	TargetPath   string    `json:"target_path,omitempty"`
	Commit       string    `json:"commit,omitempty"`
	BaseCommit   string    `json:"base_commit,omitempty"`
}

type Filter struct {
	Limit        int
	Since        time.Time
	Until        time.Time
	Tool         string
	ClientID     string
	ResultStatus string
}

type Store struct {
	mu            sync.Mutex
	db            *sql.DB
	path          string
	retentionDays int
	defaultLimit  int
}

func Open(dbPath string, retentionDays, recentLimit int) (*Store, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	if recentLimit <= 0 {
		recentLimit = 100
	}
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
	s := &Store{db: db, path: dbPath, retentionDays: retentionDays, defaultLimit: recentLimit}
	s.cleanup()
	return s, nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  tool TEXT NOT NULL,
  client_id TEXT NOT NULL DEFAULT '',
  scopes TEXT NOT NULL DEFAULT '[]',
  args_digest TEXT NOT NULL DEFAULT '',
  result_status TEXT NOT NULL,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  error TEXT,
  target_path TEXT,
  "commit" TEXT,
  base_commit TEXT
);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(ts);
`

func Digest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Append(e Entry) {
	if e.TS.IsZero() {
		e.TS = time.Now()
	}
	if e.Scopes == nil {
		e.Scopes = []string{}
	}
	scopes, _ := json.Marshal(e.Scopes)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(
		`INSERT INTO audit_log(ts,tool,client_id,scopes,args_digest,result_status,duration_ms,error,target_path,"commit",base_commit)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		formatTS(e.TS),
		e.Tool,
		e.ClientID,
		string(scopes),
		e.ArgsDigest,
		e.ResultStatus,
		e.DurationMS,
		nullIfEmpty(e.Error),
		nullIfEmpty(e.TargetPath),
		nullIfEmpty(e.Commit),
		nullIfEmpty(e.BaseCommit),
	)
	s.cleanupLocked()
}

func (s *Store) List(f Filter) []Entry {
	limit := f.Limit
	if limit <= 0 {
		limit = s.defaultLimit
	}
	args := []any{}
	var where []string
	if !f.Since.IsZero() {
		where = append(where, "ts >= ?")
		args = append(args, formatTS(f.Since))
	}
	if !f.Until.IsZero() {
		where = append(where, "ts <= ?")
		args = append(args, formatTS(f.Until))
	}
	if f.Tool != "" {
		where = append(where, "tool = ?")
		args = append(args, f.Tool)
	}
	if f.ClientID != "" {
		where = append(where, "client_id = ?")
		args = append(args, f.ClientID)
	}
	if f.ResultStatus != "" && f.ResultStatus != "all" {
		where = append(where, "result_status = ?")
		args = append(args, f.ResultStatus)
	}
	q := `SELECT ts, tool, client_id, scopes, args_digest, result_status, duration_ms, error, target_path, "commit", base_commit FROM audit_log`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY ts DESC LIMIT ?"
	args = append(args, limit)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]Entry, 0)
	for rows.Next() {
		var ts, tool, clientID, scopes, digest, status string
		var duration int
		var errMsg, target, commit, base sql.NullString
		if err := rows.Scan(&ts, &tool, &clientID, &scopes, &digest, &status, &duration, &errMsg, &target, &commit, &base); err != nil {
			continue
		}
		e := Entry{
			Tool:         tool,
			ClientID:     clientID,
			ArgsDigest:   digest,
			ResultStatus: status,
			DurationMS:   duration,
			Error:        errMsg.String,
			TargetPath:   target.String,
			Commit:       commit.String,
			BaseCommit:   base.String,
		}
		if t, err := time.Parse(tsLayout, ts); err == nil {
			e.TS = t
		} else if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			e.TS = t
		} else if t, err := time.Parse(time.RFC3339, ts); err == nil {
			e.TS = t
		}
		_ = json.Unmarshal([]byte(scopes), &e.Scopes)
		if e.Scopes == nil {
			e.Scopes = []string{}
		}
		out = append(out, e)
	}
	return out
}

func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
}

func (s *Store) cleanupLocked() {
	cutoff := formatTS(time.Now().Add(-time.Duration(s.retentionDays) * 24 * time.Hour))
	_, _ = s.db.Exec(`DELETE FROM audit_log WHERE ts < ?`, cutoff)
}

// tsLayout 固定 9 位小数，避免 RFC3339Nano 丢掉尾零后字符串排序乱掉。
const tsLayout = "2006-01-02T15:04:05.000000000Z"

func formatTS(t time.Time) string {
	return t.UTC().Format(tsLayout)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
