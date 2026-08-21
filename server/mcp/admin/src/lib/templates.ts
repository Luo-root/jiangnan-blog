import { api } from "./api";

export type Tpl = {
  id: string;
  kind: string;
  name: string;
  reason?: string;
  target_type?: string;
  operation?: string;
  section?: string;
  payload?: string;
  title?: string;
  content?: string;
  tags?: string[];
  description?: string;
  scopes?: string[];
  updated_at?: string;
};

export async function loadTemplates(kind: string): Promise<Tpl[]> {
  const list = await api<Tpl[]>("/api/templates");
  return (list || []).filter((t) => (t.kind || "proposal") === kind);
}

export function fillEmpty<T extends Record<string, unknown>>(cur: T, patch: Partial<T>): T {
  const next = { ...cur };
  for (const [k, v] of Object.entries(patch)) {
    const key = k as keyof T;
    const old = next[key];
    const empty = old == null || old === "" || (Array.isArray(old) && old.length === 0);
    if (empty && v != null && v !== "") {
      next[key] = v as T[keyof T];
    }
  }
  return next;
}
