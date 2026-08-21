import type { ReactNode } from "react";

export function Modal({ title, onClose, children, wide }: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  wide?: boolean;
}) {
  return (
    <div className="fixed inset-0 z-20 flex items-center justify-center bg-ink-1/40 p-5" onClick={onClose}>
      <div
        className={`max-h-[90vh] overflow-auto rounded-2xl border border-border bg-card p-5 ${wide ? "w-[760px]" : "w-[640px]"}`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-ink-1">{title}</h3>
          <button onClick={onClose} className="text-ink-3 hover:text-ink-1">×</button>
        </div>
        {children}
      </div>
    </div>
  );
}
