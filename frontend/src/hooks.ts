import { useEffect, useState } from "react";
import { fetchCurrencies } from "@/lib/api";
import type { PortfolioResponse } from "@/types";

export function usePortfolio(baseCurrency: string) {
  const [data, setData] = useState<PortfolioResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const es = new EventSource(
      `/api/portfolio/stream?base=${encodeURIComponent(baseCurrency)}`,
    );

    es.onmessage = (ev) => {
      try {
        setData(JSON.parse(ev.data) as PortfolioResponse);
        setError(null);
      } catch (e) {
        setError((e as Error).message);
      }
    };

    es.addEventListener("app-error", (ev) => {
      try {
        const j = JSON.parse((ev as MessageEvent<string>).data);
        setError(j?.error ?? "stream error");
      } catch {
        setError("stream error");
      }
    });

    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) setError("stream closed");
    };

    return () => es.close();
  }, [baseCurrency]);

  return { data, error, loading: data === null };
}

export function useCurrencies() {
  const [currencies, setCurrencies] = useState<string[]>([]);
  useEffect(() => {
    fetchCurrencies()
      .then(setCurrencies)
      .catch(() => setCurrencies([]));
  }, []);
  return currencies;
}
