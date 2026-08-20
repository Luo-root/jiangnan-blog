import { clearSession, loadSession, saveSession, token } from "./auth";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function parseError(r: Response): Promise<string> {
  const text = await r.text();
  try {
    const j = JSON.parse(text) as { error?: string };
    return j.error || text || r.statusText;
  } catch {
    return text || r.statusText;
  }
}

async function refreshOnce(): Promise<boolean> {
  const sess = loadSession();
  if (!sess) return false;
  const r = await fetch("/api/admin/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: sess.refresh }),
  });
  if (!r.ok) {
    clearSession();
    return false;
  }
  saveSession(await r.json());
  return true;
}

export async function api<T>(path: string, method = "GET", body?: unknown, retried = false): Promise<T> {
  const headers: Record<string, string> = {};
  const t = token();
  if (t) headers.Authorization = `Bearer ${t}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const r = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (r.status === 401 && !retried && path !== "/api/admin/login" && path !== "/api/admin/refresh") {
    if (await refreshOnce()) return api<T>(path, method, body, true);
    if (location.pathname !== "/login") location.assign("/login");
  }
  if (!r.ok) throw new ApiError(r.status, await parseError(r));
  if (r.status === 204) return undefined as T;
  return r.json() as Promise<T>;
}

export async function login(user: string, password: string) {
  const sess = await api<import("./auth").Session>("/api/admin/login", "POST", { user, password });
  saveSession(sess);
  return sess;
}

export async function logout() {
  try {
    await api("/api/admin/logout", "POST");
  } finally {
    clearSession();
  }
}
