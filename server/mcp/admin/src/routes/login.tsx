import { FormEvent, useState } from "react";
import { ApiError, login } from "../lib/api";

export function LoginPage() {
  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await login(user.trim(), password);
      location.assign("/workspace/inbox");
    } catch (err) {
      if (err instanceof ApiError && err.status === 429) setError("失败次数过多，请稍后再试。");
      else if (err instanceof ApiError && err.status === 401) setError("用户名或口令不对。");
      else setError(err instanceof Error ? err.message : "登录失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center bg-background px-6">
      <form onSubmit={onSubmit} className="w-full max-w-sm rounded-2xl border border-border bg-card p-8 shadow-sm">
        <p className="font-mono text-[11px] tracking-[0.18em] text-ink-4 uppercase">Workbase Admin</p>
        <h1 className="mt-2 text-2xl font-semibold text-ink-1">遇见江楠 · 后台</h1>
        <p className="mt-2 text-sm text-ink-3">独立登录页，session token，不用浏览器弹窗。</p>
        <label className="mt-6 block text-xs font-medium text-ink-2">
          用户
          <input
            className="mt-1.5 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none focus:border-primary"
            value={user}
            autoComplete="username"
            onChange={(e) => setUser(e.target.value)}
          />
        </label>
        <label className="mt-4 block text-xs font-medium text-ink-2">
          口令
          <input
            type="password"
            className="mt-1.5 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none focus:border-primary"
            value={password}
            autoComplete="current-password"
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}
        <button
          type="submit"
          disabled={busy || !user || !password}
          className="mt-6 w-full rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50"
        >
          {busy ? "登录中…" : "登录"}
        </button>
      </form>
    </div>
  );
}
