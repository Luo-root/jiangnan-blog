const TOKEN_KEY = "workbase.admin.token";
const REFRESH_KEY = "workbase.admin.refresh";
const USER_KEY = "workbase.admin.user";

export type Session = {
  token: string;
  refresh_token: string;
  expires_in: number;
  user: string;
};

export function loadSession(): { token: string; refresh: string; user: string } | null {
  const token = localStorage.getItem(TOKEN_KEY);
  const refresh = localStorage.getItem(REFRESH_KEY);
  const user = localStorage.getItem(USER_KEY);
  if (!token || !refresh) return null;
  return { token, refresh, user: user || "" };
}

export function saveSession(s: Session) {
  localStorage.setItem(TOKEN_KEY, s.token);
  localStorage.setItem(REFRESH_KEY, s.refresh_token);
  localStorage.setItem(USER_KEY, s.user);
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(USER_KEY);
}

export function token(): string {
  return localStorage.getItem(TOKEN_KEY) || "";
}
