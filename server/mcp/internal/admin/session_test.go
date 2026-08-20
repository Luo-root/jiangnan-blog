package admin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionStoreIssueLookupRevoke(t *testing.T) {
	s := NewSessionStore(2)
	token, refresh, exp, err := s.Issue("jiangnan")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || refresh == "" || token == refresh {
		t.Fatalf("token=%q refresh=%q", token, refresh)
	}
	if exp != 2 {
		t.Fatalf("expires_in=%d", exp)
	}
	got, ok := s.Lookup(token)
	if !ok || got.User != "jiangnan" {
		t.Fatalf("lookup: %+v ok=%v", got, ok)
	}
	s.Revoke(token)
	if _, ok := s.Lookup(token); ok {
		t.Fatal("revoked token still valid")
	}
	if _, _, _, err := s.Rotate(refresh); err == nil {
		t.Fatal("refresh after revoke should fail")
	}
}

func TestSessionStoreExpireAndRotate(t *testing.T) {
	s := NewSessionStore(1)
	s.ttl = 20 * time.Millisecond
	s.refresh = 200 * time.Millisecond
	token, refresh, _, err := s.Issue("jiangnan")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := s.Lookup(token); ok {
		t.Fatal("expired access token still valid")
	}
	newToken, newRefresh, _, err := s.Rotate(refresh)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if newToken == token || newRefresh == refresh {
		t.Fatal("rotate should issue new pair")
	}
	if _, ok := s.Lookup(token); ok {
		t.Fatal("old access should be dead after rotate")
	}
	got, ok := s.Lookup(newToken)
	if !ok || got.User != "jiangnan" {
		t.Fatalf("new token: %+v ok=%v", got, ok)
	}
}

func TestLoginLimiter(t *testing.T) {
	l := NewLoginLimiter(2, time.Minute)
	if !l.Allow("1.1.1.1") {
		t.Fatal("fresh key should allow")
	}
	l.Fail("1.1.1.1")
	l.Fail("1.1.1.1")
	if l.Allow("1.1.1.1") {
		t.Fatal("over limit should deny")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("other key should allow")
	}
	l.Clear("1.1.1.1")
	if !l.Allow("1.1.1.1") {
		t.Fatal("cleared key should allow")
	}
}

func passHash(pass string) string {
	sum := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(sum[:])
}

func testHandler() *Handler {
	return &Handler{
		AdminUser:     "jiangnan",
		AdminPassHash: passHash("secret"),
		Sessions:      NewSessionStore(3600),
		Limiter:       NewLoginLimiter(5, time.Minute),
	}
}

func doJSON(h http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestLoginSessionNoBasicAuth(t *testing.T) {
	h := testHandler()

	w := doJSON(h, http.MethodGet, "/api/inbox", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no token GET /api/inbox: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("must not send Basic WWW-Authenticate, got %q", w.Header().Get("WWW-Authenticate"))
	}

	w = doJSON(h, http.MethodPost, "/api/admin/login", loginReq{User: "jiangnan", Password: "wrong"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad password: %d", w.Code)
	}

	w = doJSON(h, http.MethodPost, "/api/admin/login", loginReq{User: "jiangnan", Password: "secret"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var sess loginResp
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.Token == "" || sess.RefreshToken == "" || sess.User != "jiangnan" || sess.ExpiresIn != 3600 {
		t.Fatalf("login body: %+v", sess)
	}

	w = doJSON(h, http.MethodGet, "/api/admin/me", nil, sess.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}

	w = doJSON(h, http.MethodGet, "/api/auth_tokens", nil, sess.Token)
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("authed request still 401: %s", w.Body.String())
	}

	w = doJSON(h, http.MethodPost, "/api/admin/refresh", refreshReq{RefreshToken: sess.RefreshToken}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", w.Code, w.Body.String())
	}
	var rotated loginResp
	if err := json.Unmarshal(w.Body.Bytes(), &rotated); err != nil {
		t.Fatal(err)
	}
	w = doJSON(h, http.MethodGet, "/api/admin/me", nil, sess.Token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("old token after refresh should 401, got %d", w.Code)
	}
	w = doJSON(h, http.MethodGet, "/api/admin/me", nil, rotated.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("new token me: %d", w.Code)
	}

	w = doJSON(h, http.MethodPost, "/api/admin/logout", nil, rotated.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: %d", w.Code)
	}
	w = doJSON(h, http.MethodGet, "/api/admin/me", nil, rotated.Token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("after logout: %d", w.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	h := testHandler()
	h.Limiter = NewLoginLimiter(2, time.Minute)
	body := loginReq{User: "jiangnan", Password: "nope"}
	if code := doJSON(h, http.MethodPost, "/api/admin/login", body, "").Code; code != http.StatusUnauthorized {
		t.Fatalf("1st fail: %d", code)
	}
	if code := doJSON(h, http.MethodPost, "/api/admin/login", body, "").Code; code != http.StatusUnauthorized {
		t.Fatalf("2nd fail: %d", code)
	}
	w := doJSON(h, http.MethodPost, "/api/admin/login", body, "")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd fail should 429, got %d %s", w.Code, w.Body.String())
	}
	w = doJSON(h, http.MethodPost, "/api/admin/login", loginReq{User: "jiangnan", Password: "secret"}, "")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("good password still 429 while limited, got %d", w.Code)
	}
}

func TestStaticLoginPagePublic(t *testing.T) {
	h := testHandler()
	for _, path := range []string{"/", "/login", "/workspace/inbox"} {
		w := doJSON(h, http.MethodGet, path, nil, "")
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s should be public, got %d", path, w.Code)
		}
		if w.Header().Get("WWW-Authenticate") != "" {
			t.Fatalf("%s must not challenge Basic Auth", path)
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("%s content-type=%q", path, ct)
		}
	}
}
