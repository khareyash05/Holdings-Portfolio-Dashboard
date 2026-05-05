import { useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatMoney, formatPct } from "@/lib/format";
import type { Holding, GroupBy } from "@/types";
import { bucketsFor } from "@/lib/buckets";

interface Props {
  holdings: Holding[];
  baseCurrency: string;
}

export function Buckets({ holdings, baseCurrency }: Props) {
  const [groupBy, setGroupBy] = useState<GroupBy>("exchange");
  const [sortKey, setSortKey] = useState<"value" | "gain">("value");

  const buckets = bucketsFor(holdings, groupBy);
  const sorted = [...buckets].sort((a, b) =>
    sortKey === "value" ? b.current - a.current : b.gainPct - a.gainPct,
  );

  return (
    <section className="mt-8">
      <div className="mb-3 flex flex-wrap items-baseline justify-between gap-3">
        <h2 className="text-sm font-semibold uppercase tracking-wider">
          Allocation
        </h2>
        <div className="flex gap-4 text-sm">
          <div className="flex items-baseline gap-2">
            <label className="text-muted-foreground">By</label>
            <Select
              value={groupBy}
              onValueChange={(v) => setGroupBy(v as GroupBy)}
            >
              <SelectTrigger className="h-7 w-[110px] rounded-none">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="exchange">Exchange</SelectItem>
                <SelectItem value="country">Country</SelectItem>
                <SelectItem value="sector">Sector</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-baseline gap-2">
            <label className="text-muted-foreground">Sort</label>
            <Select
              value={sortKey}
              onValueChange={(v) => setSortKey(v as "value" | "gain")}
            >
              <SelectTrigger className="h-7 w-[100px] rounded-none">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="value">Value</SelectItem>
                <SelectItem value="gain">Gain%</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] border border-border">
        {sorted.map((b) => {
          const cls = b.gainAbs >= 0 ? "text-pos" : "text-neg";
          return (
            <div key={b.key} className="border-b border-r border-border/50 p-3">
              <div className="flex items-baseline justify-between text-xs text-muted-foreground">
                <span className="font-medium text-foreground">{b.key}</span>
                <span>{b.count}</span>
              </div>
              <div className="mt-1.5 text-lg tabular-nums">
                {formatMoney(b.current, baseCurrency)}
              </div>
              <div className="flex justify-between text-xs tabular-nums">
                <span className={cls}>{formatPct(b.gainPct)}</span>
                <span className={cls}>
                  {formatMoney(b.gainAbs, baseCurrency)}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
