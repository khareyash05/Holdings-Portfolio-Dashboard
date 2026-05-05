export async function fetchCurrencies(): Promise<string[]> {
  const r = await fetch("/api/currencies");
  if (!r.ok) throw new Error(`currencies: ${r.status}`);
  const j = await r.json();
  return j.currencies as string[];
}
