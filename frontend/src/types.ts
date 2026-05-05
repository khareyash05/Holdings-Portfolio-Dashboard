export interface Summary {
  baseCurrency: string;
  netWorth: number;
  invested: number;
  unrealizedGains: number;
  gainsPct: number;
  stale: boolean;
}

export interface Holding {
  ticker: string;
  name: string;
  exchange: string;
  country: string;
  sector: string;
  localCurrency: string;
  quantity: number;
  currentPriceLocal: number;
  boughtPriceLocal: number;
  currentPriceBase: number;
  investedBase: number;
  netWorthBase: number;
  unrealizedGains: number;
  gainsPct: number;
  stale: boolean;
}

export interface PortfolioResponse {
  baseCurrency: string;
  asOf: string;
  summary: Summary;
  holdings: Holding[];
  exchanges?: unknown;
}

export interface Bucket {
  key: string;
  current: number;
  invested: number;
  gainAbs: number;
  gainPct: number;
  count: number;
}

export type GroupBy = "exchange" | "country" | "sector";
export type SortDir = "asc" | "desc";
export type HoldingSortKey =
  | "ticker"
  | "sector"
  | "exchange"
  | "country"
  | "quantity"
  | "netWorthBase"
  | "unrealizedGains"
  | "gainsPct";
