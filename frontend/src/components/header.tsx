import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface Props {
  baseCurrency: string;
  onChange: (c: string) => void;
  currencies: string[];
  asOf?: string;
}

const PREFERRED = [
  "INR",
  "USD",
  "EUR",
  "GBP",
  "JPY",
  "CHF",
  "CAD",
  "AUD",
  "HKD",
];

export function Header({ baseCurrency, onChange, currencies, asOf }: Props) {
  const sorted = [
    ...PREFERRED.filter((c) => currencies.includes(c)),
    ...currencies.filter((c) => !PREFERRED.includes(c)),
  ];
  return (
    <header className="mb-6 flex items-baseline justify-between border-b border-zinc-800 pb-4">
      <div className="flex items-baseline gap-3">
        <h1 className="text-lg font-semibold">Paasa Portfolio</h1>
        <p className="text-xs text-neutral-400">
          {asOf
            ? `Updated ${new Date(asOf).toLocaleTimeString()}`
            : "Loading..."}
        </p>
      </div>
      <div className="flex items-baseline gap-2 text-sm">
        <label className="text-neutral-400">Base</label>
        <Select value={baseCurrency} onValueChange={onChange}>
          <SelectTrigger className="h-8 w-[100px] rounded-none">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {sorted.map((c) => (
              <SelectItem key={c} value={c}>
                {c}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </header>
  );
}
