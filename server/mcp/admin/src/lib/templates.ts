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

export function fillEmpty<T extends Record<string, unknown>>(cur: T, patch: Partial<T>, defaults?: Partial<T>): { next: T; filled: string[] } {
  const next = { ...cur };
  const filled: string[] = [];
  for (const [k, v] of Object.entries(patch)) {
    const key = k as keyof T;
    const old = next[key];
    const def = defaults?.[key];
    const atDefault = def !== undefined && JSON.stringify(old) === JSON.stringify(def);
    const empty = old == null || old === "" || (Array.isArray(old) && old.length === 0) || atDefault;
    if (empty && v != null && v !== "" && !(Array.isArray(v) && v.length === 0)) {
      next[key] = v as T[keyof T];
      filled.push(String(k));
    }
  }
  return { next, filled };
}
