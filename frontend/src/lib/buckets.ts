import type { Bucket, GroupBy, Holding } from "@/types";

export function bucketsFor(holdings: Holding[], groupBy: GroupBy): Bucket[] {
  const acc = new Map<string, Bucket>();
  for (const h of holdings) {
    const key = h[groupBy] || "Unknown";
    let b = acc.get(key);
    if (!b) {
      b = { key, current: 0, invested: 0, gainAbs: 0, gainPct: 0, count: 0 };
      acc.set(key, b);
    }
    b.current += h.netWorthBase;
    b.invested += h.investedBase;
    b.gainAbs += h.unrealizedGains;
    b.count += 1;
  }
  for (const b of acc.values()) {
    b.gainPct = b.invested > 0 ? (b.gainAbs / b.invested) * 100 : 0;
  }
  return [...acc.values()];
}
