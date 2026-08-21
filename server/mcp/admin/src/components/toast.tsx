import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";

type Kind = "success" | "error" | "info";
type Item = { id: number; kind: Kind; text: string };

type ToastAPI = {
  success: (text: string) => void;
  error: (text: string) => void;
  info: (text: string) => void;
};

const Ctx = createContext<ToastAPI | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Item[]>([]);
  const seq = useRef(1);

  const dismiss = useCallback((id: number) => {
    setItems((xs) => xs.filter((x) => x.id !== id));
  }, []);

  const push = useCallback((kind: Kind, text: string) => {
    const id = seq.current++;
    setItems((xs) => [...xs, { id, kind, text }]);
    if (kind !== "error") {
      window.setTimeout(() => dismiss(id), 3200);
    }
  }, [dismiss]);

  const api: ToastAPI = {
    success: (text) => push("success", text),
    error: (text) => push("error", text),
    info: (text) => push("info", text),
  };

  return (
    <Ctx.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed right-4 top-4 z-50 flex w-80 flex-col gap-2">
        {items.map((t) => (
          <div
            key={t.id}
            className={`pointer-events-auto flex items-start gap-2 rounded-xl border px-3 py-2.5 text-sm shadow-sm ${
              t.kind === "success"
                ? "border-accent/40 bg-card text-ink-1"
                : t.kind === "error"
                  ? "border-destructive/40 bg-card text-ink-1"
                  : "border-border bg-card text-ink-1"
            }`}
          >
            <span className={`mt-1 h-2 w-2 shrink-0 rounded-full ${
              t.kind === "success" ? "bg-accent" : t.kind === "error" ? "bg-destructive" : "bg-primary"
            }`} />
            <p className="min-w-0 flex-1 leading-5">{t.text}</p>
            {t.kind === "error" ? (
              <button className="shrink-0 text-ink-3 hover:text-ink-1" onClick={() => dismiss(t.id)} aria-label="关闭">×</button>
            ) : null}
          </div>
        ))}
      </div>
    </Ctx.Provider>
  );
}

export function useToast(): ToastAPI {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useToast must be used inside ToastProvider");
  return ctx;
}

export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
