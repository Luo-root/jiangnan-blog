import type { ReactNode } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function AppDialog({
  title,
  open,
  onClose,
  children,
  wide,
}: {
  title: string;
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  wide?: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent className={`max-h-[90vh] overflow-auto ${wide ? "sm:max-w-[760px]" : "sm:max-w-[640px]"}`}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  );
}
