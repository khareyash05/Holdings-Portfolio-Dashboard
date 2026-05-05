import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatMoney, formatPct, formatQty } from "@/lib/format";
import type { HoldingSortKey, SortDir ,Holding} from "@/types";

interface Props {
  holdings: Holding[];
  baseCurrency: string;
}

const COLS: { key: HoldingSortKey; label: string; numeric?: boolean }[] = [
  { key: "ticker", label: "Ticker" },
  { key: "sector", label: "Sector" },
  { key: "exchange", label: "Exch" },
  { key: "quantity", label: "Qty", numeric: true },
  { key: "netWorthBase", label: "Value", numeric: true },
  { key: "unrealizedGains", label: "Gain", numeric: true },
  { key: "gainsPct", label: "Gain%", numeric: true },
];

export function Holdings({ holdings, baseCurrency }: Props) {
  const [sortKey, setSortKey] = useState<HoldingSortKey>("netWorthBase");
  const [dir, setDir] = useState<SortDir>("desc");

  const sorted = [...holdings].sort((a, b) => {
    const av = a[sortKey],
      bv = b[sortKey];
    if (typeof av === "number" && typeof bv === "number") {
      return dir === "asc" ? av - bv : bv - av;
    }
    return dir === "asc"
      ? String(av).localeCompare(String(bv))
      : String(bv).localeCompare(String(av));
  });

  function onClick(k: HoldingSortKey) {
    if (sortKey === k) setDir(dir === "desc" ? "asc" : "desc");
    else {
      setSortKey(k);
      setDir("desc");
    }
  }

  return (
    <section className="mt-8">
      <div className="mb-3 flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase tracking-wider">
          Holdings
        </h2>
        <span className="text-xs text-muted-foreground">
          {holdings.length} rows
        </span>
      </div>
      <div className="overflow-x-auto border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              {COLS.map((c) => (
                <TableHead
                  key={c.key}
                  onClick={() => onClick(c.key)}
                  className={[
                    "cursor-pointer select-none hover:text-foreground",
                    c.numeric ? "text-right" : "",
                    sortKey === c.key ? "text-foreground" : "",
                  ].join(" ")}
                >
                  {c.label}
                  {sortKey === c.key ? (dir === "desc" ? " ↓" : " ↑") : ""}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sorted.map((h) => {
              const cls = h.unrealizedGains >= 0 ? 'text-pos' : 'text-neg';
              return (
                <TableRow key={h.ticker}>
                  <TableCell className="font-mono">{h.ticker}</TableCell>
                  <TableCell>{h.sector}</TableCell>
                  <TableCell>{h.exchange}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {formatQty(h.quantity)}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {formatMoney(h.netWorthBase, baseCurrency)}
                  </TableCell>
                  <TableCell className={`text-right tabular-nums ${cls}`}>
                    {formatMoney(h.unrealizedGains, baseCurrency)}
                  </TableCell>
                  <TableCell className={`text-right tabular-nums ${cls}`}>
                    {formatPct(h.gainsPct)}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}
