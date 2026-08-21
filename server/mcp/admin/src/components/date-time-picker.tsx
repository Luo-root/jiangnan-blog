import { useMemo, useState } from "react";
import { CalendarDays } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

function pad(n: number) {
  return String(n).padStart(2, "0");
}

function hmOf(d: Date) {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function applyTime(day: Date, hm: string): Date | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec(hm.trim());
  if (!m) return null;
  const h = Number(m[1]);
  const min = Number(m[2]);
  if (h > 23 || min > 59) return null;
  const next = new Date(day);
  next.setHours(h, min, 0, 0);
  if (Number.isNaN(next.getTime())) return null;
  return next;
}

export function DateTimePicker({
  label,
  value,
  onChange,
}: {
  label: string;
  value: Date | null;
  onChange: (d: Date | null) => void;
}) {
  const [open, setOpen] = useState(false);
  const [hm, setHm] = useState(value ? hmOf(value) : "00:00");
  const text = useMemo(() => {
    if (!value) return "未选择";
    return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())} ${hmOf(value)}`;
  }, [value]);

  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-ink-3">{label}</span>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button type="button" variant="outline" size="sm" className="min-w-48 justify-start font-mono">
            <CalendarDays className="opacity-70" />
            {text}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-3" align="start">
          <Calendar
            mode="single"
            selected={value ?? undefined}
            onSelect={(d) => {
              if (!d) {
                onChange(null);
                return;
              }
              const next = applyTime(d, hm) ?? d;
              onChange(next);
            }}
          />
          <div className="mt-3 flex items-center gap-2">
            <span className="text-xs text-ink-3">时分</span>
            <Input
              className="h-8 w-24 font-mono"
              value={hm}
              placeholder="HH:mm"
              onChange={(e) => {
                const v = e.target.value;
                setHm(v);
                if (!value) return;
                const next = applyTime(value, v);
                if (next) onChange(next);
              }}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setHm("00:00");
                onChange(null);
                setOpen(false);
              }}
            >
              清除
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}

export function dateToRFC3339(d: Date | null): string | null {
  if (!d) return "";
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}
