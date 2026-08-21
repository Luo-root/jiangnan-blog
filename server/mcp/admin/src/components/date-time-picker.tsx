import { useMemo, useState } from "react";
import { CalendarDays } from "lucide-react";
import type { DateRange } from "react-day-picker";
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

function fmtDay(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function DateRangePicker({
  from,
  to,
  onChange,
}: {
  from: Date | null;
  to: Date | null;
  onChange: (from: Date | null, to: Date | null) => void;
}) {
  const [open, setOpen] = useState(false);
  const [fromHm, setFromHm] = useState(from ? hmOf(from) : "00:00");
  const [toHm, setToHm] = useState(to ? hmOf(to) : "23:59");
  const selected: DateRange | undefined = from || to ? { from: from ?? undefined, to: to ?? undefined } : undefined;
  const text = useMemo(() => {
    if (!from && !to) return "选择日期范围";
    const a = from ? `${fmtDay(from)} ${hmOf(from)}` : "…";
    const b = to ? `${fmtDay(to)} ${hmOf(to)}` : "…";
    return `${a}  →  ${b}`;
  }, [from, to]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm" className="min-w-72 justify-start font-mono font-medium">
          <CalendarDays className="opacity-70" />
          {text}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-3" align="start">
        <Calendar
          mode="range"
          numberOfMonths={2}
          selected={selected}
          onSelect={(r) => {
            const nf = r?.from ? (applyTime(r.from, fromHm) ?? r.from) : null;
            const nt = r?.to ? (applyTime(r.to, toHm) ?? r.to) : null;
            onChange(nf, nt);
          }}
        />
        <div className="mt-3 flex items-center gap-3">
          <span className="text-xs font-medium text-ink-2">从</span>
          <Input
            className="h-8 w-24 font-mono"
            value={fromHm}
            placeholder="HH:mm"
            onChange={(e) => {
              const v = e.target.value;
              setFromHm(v);
              if (!from) return;
              const next = applyTime(from, v);
              if (next) onChange(next, to);
            }}
          />
          <span className="text-xs font-medium text-ink-2">到</span>
          <Input
            className="h-8 w-24 font-mono"
            value={toHm}
            placeholder="HH:mm"
            onChange={(e) => {
              const v = e.target.value;
              setToHm(v);
              if (!to) return;
              const next = applyTime(to, v);
              if (next) onChange(from, next);
            }}
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              setFromHm("00:00");
              setToHm("23:59");
              onChange(null, null);
            }}
          >
            清除
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function dateToRFC3339(d: Date | null): string | null {
  if (!d) return "";
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}
