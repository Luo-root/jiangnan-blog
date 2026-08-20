package admin

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

const defaultRefreshTTL = 7 * 24 * time.Hour

// Session 是 webUI 登录后的短期访问凭证（设计 §21.5）。
type Session struct {
	User      string
	ExpiresAt time.Time
	Refresh   string
}

type refreshRec struct {
	User      string
	Access    string
	ExpiresAt time.Time
}

// SessionStore 内存存 session / refresh。进程重启要重新登录。
type SessionStore struct {
	mu        sync.Mutex
	ttl       time.Duration
	refresh   time.Duration
	sessions  map[string]*Session
	refreshes map[string]*refreshRec
}

func NewSessionStore(ttlSeconds int) *SessionStore {
	ttl := time.Duration(ttlSeconds) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &SessionStore{
		ttl:       ttl,
		refresh:   defaultRefreshTTL,
		sessions:  map[string]*Session{},
		refreshes: map[string]*refreshRec{},
	}
}

func (s *SessionStore) Issue(user string) (token, refresh string, expiresIn int, err error) {
	token, err = randomToken()
	if err != nil {
		return "", "", 0, err
	}
	refresh, err = randomToken()
	if err != nil {
		return "", "", 0, err
	}
	now := time.Now()
	s.mu.Lock()
	s.sessions[token] = &Session{User: user, ExpiresAt: now.Add(s.ttl), Refresh: refresh}
	s.refreshes[refresh] = &refreshRec{User: user, Access: token, ExpiresAt: now.Add(s.refresh)}
	s.mu.Unlock()
	return token, refresh, int(s.ttl.Seconds()), nil
}

func (s *SessionStore) Lookup(token string) (*Session, bool) {
	if token == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return nil, false
	}
	if !time.Now().Before(sess.ExpiresAt) {
		delete(s.sessions, token)
		return nil, false
	}
	cp := *sess
	return &cp, true
}

func (s *SessionStore) Rotate(refresh string) (token, newRefresh string, expiresIn int, err error) {
	if refresh == "" {
		return "", "", 0, errUnauthorizedSession
	}
	s.mu.Lock()
	rec, ok := s.refreshes[refresh]
	if !ok || !time.Now().Before(rec.ExpiresAt) {
		if ok {
			s.revokeLocked(rec.Access, refresh)
		}
		s.mu.Unlock()
		return "", "", 0, errUnauthorizedSession
	}
	user := rec.User
	s.revokeLocked(rec.Access, refresh)
	s.mu.Unlock()
	return s.Issue(user)
}

func (s *SessionStore) Revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[token]; ok {
		s.revokeLocked(token, sess.Refresh)
	}
}

func (s *SessionStore) revokeLocked(token, refresh string) {
	delete(s.sessions, token)
	delete(s.refreshes, refresh)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// LoginLimiter 按 key（IP）计失败次数，窗口内超过 limit 拒绝。
type LoginLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	fails  map[string][]time.Time
}

func NewLoginLimiter(limit int, window time.Duration) *LoginLimiter {
	if limit <= 0 {
		limit = 5
	}
	if window <= 0 {
		window = time.Minute
	}
	return &LoginLimiter{limit: limit, window: window, fails: map[string][]time.Time{}}
}

func (l *LoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.pruneLocked(key)) < l.limit
}

func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	hits := l.pruneLocked(key)
	l.fails[key] = append(hits, time.Now())
}

func (l *LoginLimiter) Clear(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}

func (l *LoginLimiter) pruneLocked(key string) []time.Time {
	now := time.Now()
	cutoff := now.Add(-l.window)
	hits := l.fails[key]
	n := 0
	for _, t := range hits {
		if t.After(cutoff) {
			hits[n] = t
			n++
		}
	}
	hits = hits[:n]
	if n == 0 {
		delete(l.fails, key)
		return nil
	}
	l.fails[key] = hits
	return hits
}
