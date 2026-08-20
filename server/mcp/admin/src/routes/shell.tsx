import type { ReactNode } from "react";
import { loadSession } from "../lib/auth";
import { logout } from "../lib/api";
import { navigate } from "../lib/nav";
import { InboxPage } from "./workspace/inbox";
import { ProposalsPage } from "./workspace/proposals";
import { ProposalDetailPage } from "./workspace/proposal-detail";
import { AccessPage } from "./workspace/access";
import { TokenPage } from "./settings/token";

const NAV = [
  { group: "Workspace", items: [
    { href: "/workspace/inbox", label: "待办看板" },
    { href: "/workspace/proposal", label: "Proposal" },
    { href: "/workspace/access", label: "访问热度" },
  ]},
  { group: "Settings", items: [
    { href: "/settings/token", label: "Token" },
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
  else if (path === "/settings/token") page = <TokenPage />;
  else page = <p className="text-sm text-ink-3">没有这个页面。本条 PR 落地登录 + inbox / proposal / access / token。</p>;

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
                <button
                  key={it.href}
                  onClick={() => navigate(it.href)}
                  className={`mb-0.5 flex w-full rounded-lg px-3 py-2 text-left text-[13px] ${active ? "bg-primary/10 font-semibold text-primary" : "text-ink-2 hover:bg-muted"}`}
                >
                  {it.label}
                </button>
              );
            })}
          </nav>
        ))}
        <div className="mt-auto px-2 pt-4 text-[11px] text-ink-4">
          <div className="font-mono">{user}</div>
          <button
            className="mt-2 text-ink-3 underline-offset-2 hover:text-destructive hover:underline"
            onClick={async () => { await logout(); location.assign("/login"); }}
          >
            退出
          </button>
        </div>
      </aside>
      <main className="min-w-0 flex-1 overflow-hidden bg-background">{page}</main>
    </div>
  );
}
