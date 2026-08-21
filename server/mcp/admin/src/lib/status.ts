export const PROPOSAL_BADGE: Record<string, string> = {
  pending: "border-transparent bg-slate-700 text-white hover:bg-slate-700",
  conflict: "border-transparent bg-amber-500 text-white hover:bg-amber-500",
  approved: "border-transparent bg-sky-600 text-white hover:bg-sky-600",
  applied: "border-transparent bg-emerald-600 text-white hover:bg-emerald-600",
  rejected: "border-transparent bg-red-600 text-white hover:bg-red-600",
};

export const INBOX_BADGE: Record<string, string> = {
  pending: "border-transparent bg-slate-600 text-white hover:bg-slate-600",
  reviewing: "border-transparent bg-amber-500 text-white hover:bg-amber-500",
  done: "border-transparent bg-emerald-600 text-white hover:bg-emerald-600",
  abandoned: "border-transparent bg-zinc-400 text-white hover:bg-zinc-400",
};

export const AUDIT_BADGE: Record<string, string> = {
  success: "border-transparent bg-emerald-600 text-white hover:bg-emerald-600",
  error: "border-transparent bg-red-600 text-white hover:bg-red-600",
  unauthorized: "border-transparent bg-amber-500 text-white hover:bg-amber-500",
  forbidden: "border-transparent bg-red-600 text-white hover:bg-red-600",
};
