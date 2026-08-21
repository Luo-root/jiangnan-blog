import { FormEvent, useState } from "react";
import { ApiError, login } from "../lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

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
      <Card className="w-full max-w-sm">
        <CardHeader>
          <p className="font-mono text-[11px] tracking-[0.18em] text-ink-4 uppercase">Workbase Admin</p>
          <CardTitle className="text-2xl">遇见江楠 · 后台</CardTitle>
          <CardDescription>独立登录页，session token，不用浏览器弹窗。</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit}>
            <Label className="block text-sm font-medium">
              用户
              <Input className="mt-1.5" value={user} autoComplete="username" onChange={(e) => setUser(e.target.value)} />
            </Label>
            <Label className="mt-4 block text-sm font-medium">
              口令
              <Input type="password" className="mt-1.5" value={password} autoComplete="current-password" onChange={(e) => setPassword(e.target.value)} />
            </Label>
            {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}
            <Button type="submit" disabled={busy || !user || !password} className="mt-6 w-full">
              {busy ? "登录中…" : "登录"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
