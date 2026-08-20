// Package index 用 SQLite 存 vault 镜像（notes / notes_fts / links / backlinks）。
package index

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/config"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/search"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/vault"
)

type Store struct {
	mu      sync.Mutex
	db      *sql.DB
	path    string
	index   *vault.Index
	access  map[string]int
	lastAcc map[string]time.Time
}

func Open(dbPath string) (*Store, error) {
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
		db:      db,
		path:    dbPath,
		access:  map[string]int{},
		lastAcc: map[string]time.Time{},
	}
	if err := s.loadAccess(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// New 兼容旧调用。失败会 panic；新代码用 Open。
func New(dbPath string) *Store {
	s, err := Open(dbPath)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  visibility TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  access_count INTEGER DEFAULT 0,
  last_access_at TEXT,
  frontmatter_json TEXT,
  summary TEXT
);
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  id,
  title,
  headings,
  body,
  tags
);
CREATE TABLE IF NOT EXISTS links (
  source_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  link_type TEXT NOT NULL,
  raw TEXT,
  PRIMARY KEY (source_id, target_id, link_type)
);
CREATE TABLE IF NOT EXISTS backlinks (
  source_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  context TEXT,
  PRIMARY KEY (source_id, target_id)
);
`

func (s *Store) Rebuild(vaultRoot string, excluded []string, visDefault map[string]string) error {
	idx, err := vault.Scan(vaultRoot, excluded, visDefault)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.replaceLocked(idx); err != nil {
		return err
	}
	s.index = idx
	return nil
}

func (s *Store) replaceLocked(idx *vault.Index) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	prev := map[string]accessRow{}
	rows, err := tx.Query(`SELECT id, access_count, last_access_at FROM notes`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var count int
		var last sql.NullString
		if err := rows.Scan(&id, &count, &last); err != nil {
			rows.Close()
			return err
		}
		r := accessRow{Count: count}
		if last.Valid {
			if t, err := time.Parse(time.RFC3339, last.String); err == nil {
				r.Last = t
			}
		}
		prev[id] = r
	}
	rows.Close()

	if _, err := tx.Exec(`DELETE FROM notes`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM notes_fts`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM links`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM backlinks`); err != nil {
		return err
	}

	insNote, err := tx.Prepare(`INSERT INTO notes(id,path,kind,title,visibility,updated_at,access_count,last_access_at,frontmatter_json,summary) VALUES(?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insNote.Close()
	insFTS, err := tx.Prepare(`INSERT INTO notes_fts(id,title,headings,body,tags) VALUES(?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insFTS.Close()

	s.access = map[string]int{}
	s.lastAcc = map[string]time.Time{}
	for i := range idx.Notes {
		n := idx.Notes[i]
		count := 0
		var last *time.Time
		if old, ok := prev[n.ID]; ok {
			count = old.Count
			if !old.Last.IsZero() {
				t := old.Last
				last = &t
			}
		}
		s.access[n.ID] = count
		if last != nil {
			s.lastAcc[n.ID] = *last
		}
		fm, _ := json.Marshal(n.FM)
		var lastStr interface{}
		if last != nil {
			lastStr = last.Format(time.RFC3339)
		}
		if _, err := insNote.Exec(n.ID, n.ID, n.Kind, n.Title, n.Visibility, n.UpdatedAt.Format(time.RFC3339), count, lastStr, string(fm), n.Summary); err != nil {
			return err
		}
		if _, err := insFTS.Exec(n.ID, n.Title, strings.Join(n.Headings, "\n"), n.Body, strings.Join(n.Tags, " ")); err != nil {
			return err
		}
	}

	insLink, err := tx.Prepare(`INSERT INTO links(source_id,target_id,link_type,raw) VALUES(?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insLink.Close()
	insBL, err := tx.Prepare(`INSERT INTO backlinks(source_id,target_id,context) VALUES(?,?,?)`)
	if err != nil {
		return err
	}
	defer insBL.Close()
	byID := map[string]vault.Note{}
	for i := range idx.Notes {
		byID[idx.Notes[i].ID] = idx.Notes[i]
	}
	for _, l := range idx.Links {
		if _, err := insLink.Exec(l.SourceID, l.TargetID, l.LinkType, l.Raw); err != nil {
			return err
		}
		ctx := ""
		if src, ok := byID[l.SourceID]; ok {
			ctx = vault.LinkContext(src.Body, l.Raw)
		}
		if _, err := insBL.Exec(l.SourceID, l.TargetID, ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type accessRow struct {
	Count int
	Last  time.Time
}

func (s *Store) loadAccess() error {
	rows, err := s.db.Query(`SELECT id, access_count, last_access_at FROM notes`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var count int
		var last sql.NullString
		if err := rows.Scan(&id, &count, &last); err != nil {
			return err
		}
		s.access[id] = count
		if last.Valid {
			if t, err := time.Parse(time.RFC3339, last.String); err == nil {
				s.lastAcc[id] = t
			}
		}
	}
	return rows.Err()
}

func (s *Store) Notes() []vault.Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Notes
}

func (s *Store) Projects() []vault.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Projects
}

func (s *Store) Skills() []vault.Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Skills
}

func (s *Store) MCPServers() []vault.MCPServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.MCPS
}

func (s *Store) ContextPacks() []vault.ContextPack {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Context
}

func (s *Store) NoteByID(id string) *vault.Note {
	id = vault.NormalizeID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	for i := range s.index.Notes {
		if s.index.Notes[i].ID == id {
			n := s.index.Notes[i]
			return &n
		}
	}
	return nil
}

func (s *Store) Backlinks(id string) []string {
	id = vault.NormalizeID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil || s.index.Backlinks == nil {
		return nil
	}
	return append([]string(nil), s.index.Backlinks[id]...)
}

func (s *Store) ForwardLinks(id string) []vault.Link {
	id = vault.NormalizeID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	var out []vault.Link
	for _, l := range s.index.Links {
		if l.SourceID == id {
			out = append(out, l)
		}
	}
	return out
}

func (s *Store) Hit(resourceID string) int {
	resourceID = vault.NormalizeID(resourceID)
	now := time.Now()
	s.mu.Lock()
	s.access[resourceID]++
	s.lastAcc[resourceID] = now
	count := s.access[resourceID]
	db := s.db
	s.mu.Unlock()
	_, _ = db.Exec(`UPDATE notes SET access_count=?, last_access_at=? WHERE id=?`, count, now.Format(time.RFC3339), resourceID)
	return count
}

func (s *Store) Count(resourceID string) int {
	resourceID = vault.NormalizeID(resourceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.access[resourceID]
}

func (s *Store) LastAccess(resourceID string) time.Time {
	resourceID = vault.NormalizeID(resourceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAcc[resourceID]
}

type HotEntry struct {
	ResourceID string    `json:"resource_id"`
	Count      int       `json:"count"`
	LastAccess time.Time `json:"last_access"`
	Score      float64   `json:"score"`
}

// Hot 按 SCHEMA §25 实时算分：score = count * exp(-days / half_life)。
// halfLifeDays / minScore 由调用方传入（config 有值用 config，没有用代码默认）。
func (s *Store) Hot(halfLifeDays, minScore float64) []HotEntry {
	if halfLifeDays <= 0 {
		halfLifeDays = config.DefaultAccessHalfLifeDays
	}
	if minScore <= 0 {
		minScore = config.DefaultAccessMinScore
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]HotEntry, 0, len(s.access))
	for id, c := range s.access {
		if c <= 0 {
			continue
		}
		last := s.lastAcc[id]
		score := search.Access(c, last, now, halfLifeDays, 1)
		if score < minScore {
			continue
		}
		out = append(out, HotEntry{ResourceID: id, Count: c, LastAccess: last, Score: score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ResourceID < out[j].ResourceID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func (s *Store) SaveAccess() error {
	return nil
}

func (s *Store) KindCounts() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]int{}
	if s.index == nil {
		return out
	}
	for i := range s.index.Notes {
		out[s.index.Notes[i].Kind]++
	}
	return out
}

func (s *Store) Stats() (notes, projects, skills, mcps, context int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return
	}
	return len(s.index.Notes), len(s.index.Projects), len(s.index.Skills), len(s.index.MCPS), len(s.index.Context)
}
