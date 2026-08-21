import { Toaster, toast as sonner } from "sonner";
import type { ReactNode } from "react";

type ToastAPI = {
  success: (text: string) => void;
  error: (text: string) => void;
  info: (text: string) => void;
};

const api: ToastAPI = {
  success: (text) => sonner.success(text),
  error: (text) => sonner.error(text, { duration: Infinity }),
  info: (text) => sonner.info(text),
};

export function ToastProvider({ children }: { children: ReactNode }) {
  return (
    <>
      {children}
      <Toaster position="bottom-right" richColors closeButton duration={3500} />
    </>
  );
}

export function useToast(): ToastAPI {
  return api;
}

export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
