// workbase-mcp 是「遇见江楠 · Agent Workbase」的 MCP 服务端 + WebUI 后台。
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

var excludedSections = []string{".obsidian", ".trash"}

func main() {
	cfgPath := flag.String("config", "", "config.yaml 路径（默认 ./config.yaml）")
	reindexOnly := flag.Bool("reindex", false, "仅重建索引后退出")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	for _, dir := range cfg.RuntimeDirs() {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("mkdir runtime %s: %v", dir, err)
		}
	}

	inboxStore, err := inbox.New(cfg.InboxDir())
	if err != nil {
		log.Fatalf("init inbox: %v", err)
	}
	defer inboxStore.Close()

	proposalStore, err := proposal.New(cfg.ProposalsDir())
	if err != nil {
		log.Fatalf("init proposal: %v", err)
	}

	auditStore := audit.New(cfg.AuditFile(), 2000)
	idx := index.New(cfg.IndexFile())

	tokenStore, err := auth.Open(cfg.AuthDB(), cfg.Auth.GracePeriodHours)
	if err != nil {
		log.Fatalf("init auth: %v", err)
	}
	defer tokenStore.Close()

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

	mcpSrv := server.NewMCPServer(
		"workbase-mcp",
		config.IdentityVersion,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)
	tools.Register(mcpSrv, tools.Deps{
		Idx:          idx,
		Inbox:        inboxStore,
		Proposal:     proposalStore,
		Audit:        auditStore,
		Cfg:          cfg,
		VaultRoot:    cfg.Vault.Root,
		WorkbaseRoot: cfg.Workbase.Root,
		GitDir:       cfg.Vault.GitDir,
	})
	mcpSrv.Use(authAuditMiddleware(tokenStore, auditStore))

	mcpHTTP := server.NewStreamableHTTPServer(
		mcpSrv,
		server.WithEndpointPath("/mcp"),
		server.WithDisableLocalhostProtection(true),
	)

	mcpMux := http.NewServeMux()
	mcpMux.Handle("/mcp", bearerHTTPAuth(tokenStore, mcpHTTP))
	mcpMux.HandleFunc("/internal/reindex", handleInternalReindex(cfg, idx))
	mcpHTTPServer := &http.Server{Addr: cfg.Server.Listen, Handler: mcpMux}

	go func() {
		log.Printf("MCP endpoint listening on http://%s/mcp", cfg.Server.Listen)
		if err := mcpHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("mcp server: %v", err)
		}
	}()

	adminHandler := &admin.Handler{
		Inbox:            inboxStore,
		Proposal:         proposalStore,
		Index:            idx,
		Tokens:           tokenStore,
		AdminUser:        cfg.AdminAuth.User,
		AdminPassHash:    cfg.AdminAuth.PassHash,
		VaultRoot:        cfg.Vault.Root,
		GitDir:           cfg.Vault.GitDir,
		ExcludedSections: excludedSections,
		RebuildCmd:       cfg.Workbase.RebuildCmd,
	}
	adminSrv := &http.Server{Addr: cfg.Admin.Listen, Handler: adminHandler}

	go func() {
		log.Printf("admin webUI listening on http://%s", cfg.Admin.Listen)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("admin server: %v", err)
		}
	}()

	log.Printf("workbase-mcp started. Vault=%s Runtime=%s", cfg.Vault.Root, cfg.Workbase.Runtime)

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

func handleInternalReindex(cfg *config.Config, idx *index.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Printf("reindex triggered")
		if err := idx.Rebuild(cfg.Vault.Root, excludedSections); err != nil {
			log.Printf("reindex failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"notes":%d,"projects":%d,"skills":%d,"mcps":%d,"context":%d}`,
			len(idx.Notes()), len(idx.Projects()), len(idx.Skills()), len(idx.MCPServers()), len(idx.ContextPacks()))
	}
}

func bearerHTTPAuth(a *auth.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ac, err := a.AuthenticateHeader(r.Header)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer realm="workbase-mcp"`)
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithAuth(r.Context(), ac)))
	})
}

func authAuditMiddleware(a *auth.Store, ad *audit.Store) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolName := req.Params.Name
			scope := tools.RequiredScope(toolName)

			ac, ok := auth.FromContext(ctx)
			if !ok {
				var err error
				ac, err = a.AuthenticateHeader(req.Header)
				if err != nil {
					return mcp.NewToolResultError("unauthorized: missing or invalid bearer token"), nil
				}
				ctx = auth.WithAuth(ctx, ac)
			}
			if scope != "" && !ac.HasScope(scope) {
				return mcp.NewToolResultErrorf("forbidden: tool %q requires scope %q", toolName, scope), nil
			}

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
			ad.Append(toolName, scope, ac.ClientID, targetID, raw)
			return next(ctx, req)
		}
	}
}
