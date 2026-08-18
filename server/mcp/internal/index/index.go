// Package index 管理 JSON 索引与访问热度（access_count）。
package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/vault"
)

// Store 持有一个 vault 索引 + 访问计数。
type Store struct {
	mu      sync.Mutex
	path    string
	index   *vault.Index
	access  map[string]int       // resource_id -> 访问次数
	lastAcc map[string]time.Time // resource_id -> 最近访问时间
}

// New 创建 Store（懒加载 index，访问计数从磁盘恢复）。
func New(indexPath string) *Store {
	s := &Store{
		path:    indexPath,
		access:  map[string]int{},
		lastAcc: map[string]time.Time{},
	}
	s.loadAccess()
	return s
}

// Rebuild 从 vault 根重建索引并写盘。
func (s *Store) Rebuild(vaultRoot string, excluded []string) error {
	idx, err := vault.Scan(vaultRoot, excluded)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.index = idx
	s.mu.Unlock()
	return s.saveIndex()
}

// Notes 返回当前索引的 notes（无则空）。
func (s *Store) Notes() []vault.Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Notes
}

// Projects 返回项目列表。
func (s *Store) Projects() []vault.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Projects
}

// Skills 返回 skill 列表。
func (s *Store) Skills() []vault.Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Skills
}

// MCPServers 返回 mcp 列表。
func (s *Store) MCPServers() []vault.MCPServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.MCPS
}

// ContextPacks 返回 context pack 列表。
func (s *Store) ContextPacks() []vault.ContextPack {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	return s.index.Context
}

// Backlinks 返回 note 的反向链接列表。
func (s *Store) Backlinks(id string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil || s.index.Backlinks == nil {
		return nil
	}
	return s.index.Backlinks[id]
}

// Hit 记录一次访问，返回累计次数。
func (s *Store) Hit(resourceID string) int {
	s.mu.Lock()
	s.access[resourceID]++
	s.lastAcc[resourceID] = time.Now()
	count := s.access[resourceID]
	s.mu.Unlock()
	// 访问热度是运行时状态，立即落盘，避免服务重启丢失。
	_ = s.SaveAccess()
	return count
}

// Count 返回某资源的累计访问次数（不递增）。
func (s *Store) Count(resourceID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.access[resourceID]
}

// HotEntry 是热度排序条目。
type HotEntry struct {
	ResourceID string    `json:"resource_id"`
	Count      int       `json:"count"`
	LastAccess time.Time `json:"last_access"`
}

// Hot 返回按访问次数降序的热度列表。
func (s *Store) Hot() []HotEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]HotEntry, 0, len(s.access))
	for id, c := range s.access {
		out = append(out, HotEntry{ResourceID: id, Count: c, LastAccess: s.lastAcc[id]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

// ---------------------------------------------------------------------------
// 持久化（访问计数单独存 .workbase/index/access.json）
// ---------------------------------------------------------------------------

type accessFile struct {
	Access map[string]int    `json:"access"`
	Last   map[string]string `json:"last_access"`
}

func (s *Store) accessPath() string {
	return filepath.Join(filepath.Dir(s.path), "access.json")
}

func (s *Store) saveIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	os.MkdirAll(filepath.Dir(s.path), 0755)
	b, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *Store) loadAccess() {
	b, err := os.ReadFile(s.accessPath())
	if err != nil {
		return
	}
	var f accessFile
	if json.Unmarshal(b, &f) != nil {
		return
	}
	s.access = f.Access
	for id, ts := range f.Last {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			s.lastAcc[id] = t
		}
	}
}

// SaveAccess 把访问计数写盘（进程退出前调用）。
func (s *Store) SaveAccess() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := accessFile{Access: s.access, Last: map[string]string{}}
	for id, t := range s.lastAcc {
		f.Last[id] = t.Format(time.RFC3339)
	}
	os.MkdirAll(filepath.Dir(s.accessPath()), 0755)
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.accessPath(), b, 0644)
}
