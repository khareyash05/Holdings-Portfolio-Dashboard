import { useState } from "react";
import { Header } from "@/components/header";
import { SummaryCards } from "@/components/summary";
import { Buckets } from "@/components/buckets";
import { Holdings } from "@/components/holdings";
import { useCurrencies, usePortfolio } from "@/hooks";

export function App() {
  const [baseCurrency, setBaseCurrency] = useState("INR");
  const currencies = useCurrencies();
  const { data, error, loading } = usePortfolio(baseCurrency);

  return (
    <div className="mx-auto max-w-5xl px-6 pb-16 pt-8">
      <Header
        baseCurrency={baseCurrency}
        onChange={setBaseCurrency}
        currencies={currencies}
        asOf={data?.asOf}
      />

      {error && (
        <div className="mb-4 border border-red-400 px-3 py-2 text-sm text-red-400">
          {error}
        </div>
      )}

      {loading || !data ? (
        <p className="py-8 text-neutral-400">Loading...</p>
      ) : (
        <>
          <SummaryCards
            summary={data.summary}
            baseCurrency={data.baseCurrency}
          />
          <Buckets
            holdings={data.holdings}
            baseCurrency={data.baseCurrency}
          />
          <Holdings holdings={data.holdings} baseCurrency={data.baseCurrency} />
        </>
      )}
    </div>
  );
}
