import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from "recharts";
import { formatMoney, formatPct } from "@/lib/format";
import type { Bucket } from "@/types";

interface Props {
  buckets: Bucket[];
  baseCurrency: string;
}

const COLORS = [
  "red",
  "blue",
  "green",
  "orange",
  "purple",
  "yellow",
  "teal",
  "pink",
  "brown",
  "cyan",
];

export function Chart({ buckets, baseCurrency }: Props) {
  const total = buckets.reduce((s, b) => s + b.current, 0);
  const data = buckets.map((b) => ({
    name: b.key,
    value: b.current,
    pct: total > 0 ? (b.current / total) * 100 : 0,
  }));

  return (
    <div className="h-[280px] w-full border border-zinc-800 p-3">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            innerRadius={60}
            outerRadius={100}
            paddingAngle={1}
            stroke="white"
            strokeWidth={2}
          >
            {data.map((_, i) => (
              <Cell key={i} fill={COLORS[i % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{
              background: "white",
              border: "1px solid lightgray",
              borderRadius: 0,
              fontSize: 12,
            }}
            formatter={(value: number, name, item) => [
              `${formatMoney(value, baseCurrency)} (${formatPct(item.payload.pct)})`,
              item.payload.name,
            ]}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
