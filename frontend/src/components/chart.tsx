import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from "recharts";
import { formatMoney, formatPct } from "@/lib/format";
import type { Bucket } from "@/types";

interface Props {
  buckets: Bucket[];
  baseCurrency: string;
}

const COLORS = [
  "#5eb87a",
  "#7ab8c4",
  "#c4a85e",
  "#b87a7a",
  "#9a8fb8",
  "#5a8a9a",
  "#b8a85e",
  "#8a8a8a",
  "#7ab85e",
  "#b87aa8",
];

export function AllocationChart({ buckets, baseCurrency }: Props) {
  const total = buckets.reduce((s, b) => s + b.current, 0);
  const data = buckets.map((b) => ({
    name: b.key,
    value: b.current,
    pct: total > 0 ? (b.current / total) * 100 : 0,
  }));

  return (
    <div className="h-[280px] w-full border border-border p-3">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            innerRadius={60}
            outerRadius={100}
            paddingAngle={1}
            stroke="hsl(var(--background))"
            strokeWidth={2}
          >
            {data.map((_, i) => (
              <Cell key={i} fill={COLORS[i % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{
              background: "hsl(var(--background))",
              border: "1px solid hsl(var(--border))",
              borderRadius: 0,
              fontSize: 12,
            }}
            formatter={(value: number, _name, item) => [
              `${formatMoney(value, baseCurrency)} (${formatPct(item.payload.pct)})`,
              item.payload.name,
            ]}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
