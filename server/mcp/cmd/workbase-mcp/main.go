// workbase-mcp 是「遇见江楠 · Agent Workbase」的 MCP 服务端 + WebUI 后台。
//
// v0.1 范围：
//   - 从 Obsidian Vault 构建 JSON 索引（notes/projects/skills/mcps/context）
//   - inbox 待办管理（append/update/list/get + 7 天清理）
//   - proposal 写入请求（create/list/get）
//   - admin WebUI（看板式 inbox + 热度可视化）
//   - 访问计数（access_count）
//   - MCP Streamable HTTP 接入（127.0.0.1:8787/mcp）+ Bearer/scope/audit 中间件
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/admin"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/audit"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/auth"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/config"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/inbox"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/index"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/proposal"
	"github.com/Luo-root/jiangnan-blog/mcp/internal/tools"
)

// excludedSections 是 MCP 索引器排除的目录。
//
// 注意与 vite.config.ts 的差异：博客构建需要排除 Workbase/（不作为公开栏目），
// 但 MCP 索引器必须扫描 Workbase/（skill/mcp/context registry 的来源）。
var excludedSections = []string{".obsidian", ".trash"}

func main() {
	cfgPath := flag.String("config", "", "config.yaml 路径（可选）")
	reindexOnly := flag.Bool("reindex", false, "仅重建索引后退出")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 1. inbox / proposal / audit store
	inboxStore, err := inbox.New(cfg.Workbase.Inbox)
	if err != nil {
		log.Fatalf("init inbox: %v", err)
	}
	defer inboxStore.Close()

	proposalStore, err := proposal.New(cfg.Workbase.Proposals)
	if err != nil {
		log.Fatalf("init proposal: %v", err)
	}

	auditStore := audit.New(cfg.Workbase.Audit, 2000)

	// 2. 索引 + 访问计数
	idx := index.New(cfg.Workbase.Index + "/notes.json")

	// 3. 鉴权（admin 后台 + MCP）
	mcpAuth := auth.New(cfg.Auth)

	// 4. 重建索引
	log.Printf("rebuilding index from vault: %s", cfg.Vault.Root)
	if err := idx.Rebuild(cfg.Vault.Root, excludedSections); err != nil {
		log.Printf("WARN: rebuild index: %v", err)
	} else {
		log.Printf("indexed %d notes, %d projects, %d skills, %d mcps, %d context packs",
			len(idx.Notes()), len(idx.Projects()), len(idx.Skills()), len(idx.MCPServers()), len(idx.ContextPacks()))
	}

	if *reindexOnly {
		_ = idx.SaveAccess()
		log.Println("reindex done")
		return
	}

	// 5. MCP server（Streamable HTTP）
	mcpSrv := server.NewMCPServer(
		"workbase-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)
	tools.Register(mcpSrv, tools.Deps{
		Idx:       idx,
		Inbox:     inboxStore,
		Proposal:  proposalStore,
		Audit:     auditStore,
		MCPAuth:   mcpAuth,
		VaultRoot: cfg.Vault.Root,
		GitDir:    cfg.Vault.GitDir,
	})
	mcpSrv.Use(authAuditMiddleware(mcpAuth, auditStore))

	mcpHTTP := server.NewStreamableHTTPServer(
		mcpSrv,
		server.WithEndpointPath("/mcp"),
		// 本地直连 + Caddy 反代（保留 Host 头）场景都需要放行
		server.WithDisableLocalhostProtection(true),
	)

	// HTTP 层 Bearer 鉴权（§23.1 #1：未带 token → 401）+ /internal/reindex
	mcpMux := http.NewServeMux()
	mcpMux.Handle("/mcp", bearerHTTPAuth(mcpAuth, mcpHTTP))
	mcpMux.HandleFunc("/internal/reindex", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// 仅本机可触发
		if host := r.RemoteAddr; !(strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "[::1]")) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		log.Printf("reindex triggered by %s", r.RemoteAddr)
		if err := idx.Rebuild(cfg.Vault.Root, excludedSections); err != nil {
			log.Printf("reindex failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"notes":%d,"projects":%d,"skills":%d,"mcps":%d,"context":%d}`,
			len(idx.Notes()), len(idx.Projects()), len(idx.Skills()), len(idx.MCPServers()), len(idx.ContextPacks()))
	})
	mcpHTTPServer := &http.Server{Addr: cfg.Server.Listen, Handler: mcpMux}

	go func() {
		log.Printf("MCP endpoint listening on http://%s/mcp", cfg.Server.Listen)
		if err := mcpHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("mcp server: %v", err)
		}
	}()

	// 6. admin WebUI 后台
	adminHandler := &admin.Handler{Inbox: inboxStore, Proposal: proposalStore, Index: idx, AdminAuth: mcpAuth, AdminUser: cfg.Admin.Auth.User, AdminPassHash: cfg.Admin.Auth.PassHash, VaultRoot: cfg.Vault.Root, GitDir: cfg.Vault.GitDir, ExcludedSections: excludedSections, RebuildCmd: cfg.Workbase.RebuildCmd}
	adminSrv := &http.Server{Addr: cfg.Admin.Listen, Handler: adminHandler}

	go func() {
		log.Printf("admin webUI listening on http://%s", cfg.Admin.Listen)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("admin server: %v", err)
		}
	}()

	log.Printf("workbase-mcp started. Vault=%s Inbox=%s Proposals=%s Audit=%s",
		cfg.Vault.Root, cfg.Workbase.Inbox, cfg.Workbase.Proposals, cfg.Workbase.Audit)
	if mcpAuth.HasClient() {
		log.Printf("MCP auth: %d client(s) configured, bearer token + scope enforced", len(cfg.Auth.Clients))
	} else {
		log.Printf("MCP auth: DISABLED (no clients configured) — 仅限本地开发")
	}

	// 7. 优雅退出，保存访问计数
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = mcpHTTPServer.Shutdown(shutdownCtx)
	_ = adminSrv.Shutdown(shutdownCtx)
	if err := idx.SaveAccess(); err != nil {
		log.Printf("WARN: save access: %v", err)
	}
	log.Println("bye")
}

// bearerHTTPAuth 在 HTTP 层强制 Bearer Token（§23.1 #1：未带 token → 401）。
// 未配置 client 时（开发模式）直接放行。
func bearerHTTPAuth(a *auth.Auth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight 不带 Authorization，直接放行
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if a.HasClient() {
			if _, ok := a.ClientFromRequest(r); !ok {
				w.Header().Set("WWW-Authenticate", `Bearer realm="workbase-mcp"`)
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// authAuditMiddleware 做 Bearer Token + scope 校验，并记录审计。
//
// 未配置 client 时（开发模式）跳过鉴权，只记录审计。
func authAuditMiddleware(a *auth.Auth, ad *audit.Store) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolName := req.Params.Name
			scope := tools.RequiredScope(toolName)
			clientID := ""

			if a.HasClient() {
				client, ok := a.ClientFromHeader(req.Header)
				if !ok {
					return mcp.NewToolResultError("unauthorized: missing or invalid bearer token"), nil
				}
				if scope != "" && !auth.HasScope(client, scope) {
					return mcp.NewToolResultErrorf("forbidden: tool %q requires scope %q", toolName, scope), nil
				}
				clientID = client.ID
			}

			// 审计：op + scope + 目标 id + 内容哈希（不含正文）
			raw := string(req.Params.RawArguments)
			if raw == "" {
				if b, err := json.Marshal(req.Params.Arguments); err == nil {
					raw = string(b)
				}
			}
			targetID := req.GetString("id", "")
			if targetID == "" {
				targetID = req.GetString("query", "")
			}
			ad.Append(toolName, scope, clientID, targetID, raw)

			return next(ctx, req)
		}
	}
}
