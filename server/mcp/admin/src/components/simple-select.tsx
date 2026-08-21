import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

const EMPTY = "__empty";

export function SimpleSelect({
  value,
  onValue,
  items,
  className,
}: {
  value: string;
  onValue: (v: string) => void;
  items: { value: string; label: string }[];
  className?: string;
}) {
  return (
    <Select value={value || EMPTY} onValueChange={(v) => onValue(v === EMPTY ? "" : v)}>
      <SelectTrigger className={className}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="z-[80]">
        {items.map((it) => (
          <SelectItem key={it.value || EMPTY} value={it.value || EMPTY}>
            {it.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
