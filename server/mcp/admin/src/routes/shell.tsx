import type { ReactNode } from "react";
import { loadSession } from "../lib/auth";
import { logout } from "../lib/api";
import { navigate } from "../lib/nav";
import { Button } from "@/components/ui/button";
import { InboxPage } from "./workspace/inbox";
import { ProposalsPage } from "./workspace/proposals";
import { ProposalDetailPage } from "./workspace/proposal-detail";
import { AccessPage } from "./workspace/access";
import { AuditPage } from "./workspace/audit";
import { SearchPage } from "./workspace/search";
import { TokenPage } from "./settings/token";
import { SystemPage } from "./settings/system";
import { GitPage } from "./settings/git";
import { TemplatesPage } from "./settings/templates";

const NAV = [
  { group: "Workspace", items: [
    { href: "/workspace/inbox", label: "待办看板" },
    { href: "/workspace/proposal", label: "Proposal" },
    { href: "/workspace/access", label: "访问热度" },
    { href: "/workspace/audit", label: "审计日志" },
    { href: "/workspace/search", label: "知识搜索" },
  ]},
  { group: "Settings", items: [
    { href: "/settings/token", label: "Token" },
    { href: "/settings/system", label: "System" },
    { href: "/settings/git", label: "Git" },
    { href: "/settings/templates", label: "模板" },
  ]},
];

export function Shell({ path }: { path: string }) {
  const user = loadSession()?.user || "";
  const detailMatch = path.match(/^\/workspace\/proposal\/([^/]+)$/);

  let page: ReactNode;
  if (path === "/workspace/inbox") page = <InboxPage />;
  else if (path === "/workspace/proposal") page = <ProposalsPage />;
  else if (detailMatch) page = <ProposalDetailPage id={decodeURIComponent(detailMatch[1])} />;
  else if (path === "/workspace/access") page = <AccessPage />;
  else if (path === "/workspace/audit") page = <AuditPage />;
  else if (path === "/workspace/search") page = <SearchPage />;
  else if (path === "/settings/token") page = <TokenPage />;
  else if (path === "/settings/system") page = <SystemPage />;
  else if (path === "/settings/git") page = <GitPage />;
  else if (path === "/settings/templates") page = <TemplatesPage />;
  else page = <p className="text-sm text-ink-3">没有这个页面。</p>;

  return (
    <div className="flex h-full min-w-[960px]">
      <aside className="flex w-56 shrink-0 flex-col border-r border-border bg-card px-3 py-5">
        <div className="px-2 pb-4">
          <h1 className="text-base font-semibold">Workbase · <span className="text-primary">后台</span></h1>
          <p className="mt-1 font-mono text-[10px] text-ink-3">Blog as Agent Workbase</p>
        </div>
        {NAV.map((g) => (
          <nav key={g.group} className="mt-3">
            <div className="px-2 pb-1 font-mono text-[10px] uppercase tracking-wider text-ink-4">{g.group}</div>
            {g.items.map((it) => {
              const active = path === it.href || (it.href === "/workspace/proposal" && path.startsWith("/workspace/proposal/"));
              return (
                <Button
                  key={it.href}
                  variant={active ? "secondary" : "ghost"}
                  className={`mb-0.5 w-full justify-start ${active ? "font-semibold text-primary" : "text-ink-2"}`}
                  onClick={() => navigate(it.href)}
                >
                  {it.label}
                </Button>
              );
            })}
          </nav>
        ))}
        <div className="mt-auto px-2 pt-4 text-[11px] text-ink-4">
          <div className="font-mono">{user}</div>
          <Button
            variant="ghost"
            size="sm"
            className="mt-2 px-0 text-ink-3 hover:text-destructive"
            onClick={async () => { await logout(); location.assign("/login"); }}
          >
            退出
          </Button>
        </div>
      </aside>
      <main className="min-w-0 flex-1 overflow-hidden bg-background">{page}</main>
    </div>
  );
}
