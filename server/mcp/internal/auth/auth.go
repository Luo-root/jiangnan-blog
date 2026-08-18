// Package auth 做 Bearer Token + scope 校验。
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/config"
)

// Auth 持有 client 列表，负责校验请求。
type Auth struct {
	clients []config.Client
	byHash  map[string]config.Client
}

func New(cfg config.AuthConfig) *Auth {
	a := &Auth{byHash: map[string]config.Client{}}
	for _, c := range cfg.Clients {
		a.clients = append(a.clients, c)
		if c.TokenHash != "" {
			a.byHash[c.TokenHash] = c
		}
	}
	return a
}

// HasClient 是否配置了任何 client（空配置 = 无鉴权，仅开发用）。
func (a *Auth) HasClient() bool { return len(a.clients) > 0 }

// ClientFromRequest 从 Authorization header 提取 client。
func (a *Auth) ClientFromRequest(r *http.Request) (config.Client, bool) {
	return a.ClientFromHeader(r.Header)
}

// ClientFromHeader 从 header 提取 client（供 MCP 中间件等无 *http.Request 的场景使用）。
func (a *Auth) ClientFromHeader(h http.Header) (config.Client, bool) {
	hv := h.Get("Authorization")
	if !strings.HasPrefix(hv, "Bearer ") {
		return config.Client{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(hv, "Bearer "))
	hash := HashToken(token)
	c, ok := a.byHash[hash]
	return c, ok
}

// HashToken 计算 token 的 SHA-256 十六进制。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// HasScope 判断 client 是否持有某 scope。
func HasScope(c config.Client, scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
