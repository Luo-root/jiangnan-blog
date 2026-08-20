package admin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
)

var errUnauthorizedSession = errors.New("unauthorized")

type loginReq struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type loginResp struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         string `json:"user"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	key := clientIP(r)
	if h.Limiter != nil && !h.Limiter.Allow(key) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate_limited"})
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if !h.validPassword(req.User, req.Password) {
		if h.Limiter != nil {
			h.Limiter.Fail(key)
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h.Sessions == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session store not configured"})
		return
	}
	token, refresh, exp, err := h.Sessions.Issue(req.User)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if h.Limiter != nil {
		h.Limiter.Clear(key)
	}
	writeJSON(w, http.StatusOK, loginResp{
		Token:        token,
		RefreshToken: refresh,
		ExpiresIn:    exp,
		User:         req.User,
	})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if h.Sessions == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	token, refresh, exp, err := h.Sessions.Rotate(req.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	user := ""
	if sess, ok := h.Sessions.Lookup(token); ok {
		user = sess.User
	}
	writeJSON(w, http.StatusOK, loginResp{
		Token:        token,
		RefreshToken: refresh,
		ExpiresIn:    exp,
		User:         user,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := bearerToken(r.Header); ok && h.Sessions != nil {
		h.Sessions.Revoke(token)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.sessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user": user})
}

func (h *Handler) validPassword(user, pass string) bool {
	if h.AdminUser == "" || h.AdminPassHash == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(user), []byte(h.AdminUser)) != 1 {
		return false
	}
	sum := sha256.Sum256([]byte(pass))
	hash := hex.EncodeToString(sum[:])
	if len(hash) != len(h.AdminPassHash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(h.AdminPassHash)) == 1
}

func (h *Handler) sessionUser(r *http.Request) (string, bool) {
	if h.Sessions == nil {
		return "", false
	}
	token, ok := bearerToken(r.Header)
	if !ok {
		return "", false
	}
	sess, ok := h.Sessions.Lookup(token)
	if !ok {
		return "", false
	}
	return sess.User, true
}

func bearerToken(header http.Header) (string, bool) {
	v := header.Get("Authorization")
	if !strings.HasPrefix(v, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
	return token, token != ""
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isPublicAPI(path, method string) bool {
	if path == "api/admin/login" && method == http.MethodPost {
		return true
	}
	if path == "api/admin/refresh" && method == http.MethodPost {
		return true
	}
	if path == "api/admin/logout" && method == http.MethodPost {
		return true
	}
	return false
}
