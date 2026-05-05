import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMoney, formatPct } from "@/lib/format";
import type { Summary } from "@/types";

interface Props {
  summary: Summary;
  baseCurrency: string;
}

const cardCls = "rounded-none border-0 bg-transparent shadow-none";
const labelCls =
  "text-xs font-normal uppercase tracking-wider text-muted-foreground";

export function SummaryCards({ summary, baseCurrency }: Props) {
  const cls = summary.unrealizedGains >= 0 ? "text-pos" : "text-neg";
  return (
    <div className="mb-8 grid grid-cols-1 border border-border md:grid-cols-3">
      <Card className={`${cardCls} border-r border-border md:border-r`}>
        <CardHeader className="pb-2">
          <CardTitle className={labelCls}>Current</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-2xl tabular-nums">
            {formatMoney(summary.netWorth, baseCurrency)}
          </p>
        </CardContent>
      </Card>
      <Card className={`${cardCls} border-r border-border md:border-r`}>
        <CardHeader className="pb-2">
          <CardTitle className={labelCls}>Invested</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-2xl tabular-nums">
            {formatMoney(summary.invested, baseCurrency)}
          </p>
        </CardContent>
      </Card>
      <Card className={cardCls}>
        <CardHeader className="pb-2">
          <CardTitle className={labelCls}>Gain</CardTitle>
        </CardHeader>
        <CardContent>
          <p className={`text-2xl tabular-nums ${cls}`}>
            {formatMoney(summary.unrealizedGains, baseCurrency)}
          </p>
          <p className={`mt-1 text-sm tabular-nums ${cls}`}>
            {formatPct(summary.gainsPct)}
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
