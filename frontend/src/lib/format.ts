const SYMBOLS: Record<string, string> = {
  INR: "₹",
  USD: "$",
  EUR: "€",
  GBP: "£",
  JPY: "¥",
  CHF: "CHF ",
  CAD: "C$",
  AUD: "A$",
  HKD: "HK$",
  SGD: "S$",
};

export function symbolFor(code: string): string {
  return SYMBOLS[code.toUpperCase()] ?? code + " ";
}

export function formatMoney(amount: number, currency: string): string {
  const sym = symbolFor(currency);
  const sign = amount < 0 ? "-" : "";
  const abs = Math.abs(amount).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  return `${sign}${sym}${abs}`;
}

export function formatPct(pct: number): string {
  const sign = pct >= 0 ? "+" : "";
  return `${sign}${pct.toFixed(2)}%`;
}

export function formatQty(q: number): string {
  return q.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  });
}
