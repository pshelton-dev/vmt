import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api, type Meta } from "../lib/api";
import { money } from "../lib/format";
import BarChart from "../components/BarChart";
import { Card } from "../components/form";

interface ReportRow {
  id: number;
  label: string;
  total: number;
  pct: number;
}

interface Reports {
  total_cost: number;
  year_cost: number;
  avg_per_service: number;
  service_count: number;
  by_category: ReportRow[];
  by_vehicle: ReportRow[];
  monthly: { month: string; label: string; total: number }[];
}

export default function ReportsPage() {
  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });
  const q = useQuery({ queryKey: ["reports"], queryFn: () => api.get<Reports>("/reports") });

  if (q.isLoading) return <p className="text-muted">Loading…</p>;
  if (q.isError || !q.data) return <p className="text-danger">Could not load reports.</p>;

  const r = q.data;
  const currency = meta.data?.currency ?? "$";
  const fmt = (v: number) => money(currency, v);
  const hasMonthly = r.monthly.some((m) => m.total > 0);

  const stats = [
    { label: "All-time spend", value: fmt(r.total_cost) },
    { label: "This year", value: fmt(r.year_cost) },
    { label: "Avg / service", value: fmt(r.avg_per_service) },
    { label: "Records", value: String(r.service_count) },
  ];

  const shareList = (rows: ReportRow[], linkVehicles: boolean) =>
    rows.length === 0 ? (
      <p className="py-4 text-center text-muted">No data yet.</p>
    ) : (
      <ul className="space-y-2.5">
        {rows.map((row) => (
          <li key={`${row.id}-${row.label}`}>
            <div className="mb-1 flex items-baseline justify-between gap-3 text-sm">
              <span className="truncate">
                {linkVehicles ? (
                  <Link to={`/vehicles/${row.id}`} className="text-primary hover:underline">
                    {row.label}
                  </Link>
                ) : (
                  row.label
                )}
              </span>
              <span className="shrink-0 tabular-nums">
                {fmt(row.total)} <span className="text-muted">· {row.pct}%</span>
              </span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-surface-2">
              <div className="h-full rounded-full bg-primary" style={{ width: `${row.pct}%` }} />
            </div>
          </li>
        ))}
      </ul>
    );

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">Cost reports</h1>
        <a
          href="/api/v1/export/services.csv"
          download
          className="flex min-h-11 items-center rounded-lg border border-border px-4 text-sm hover:border-muted"
        >
          ⬇ Export all (CSV)
        </a>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {stats.map((s) => (
          <div key={s.label} className="rounded-xl border border-border bg-surface p-4">
            <div className="truncate text-xl font-bold md:text-2xl">{s.value}</div>
            <div className="text-sm text-muted">{s.label}</div>
          </div>
        ))}
      </div>

      <Card title="Monthly spending (last 12 months)">
        {hasMonthly ? (
          <BarChart bars={r.monthly.map((m) => ({ label: m.label, value: m.total }))} format={fmt} />
        ) : (
          <p className="py-4 text-center text-muted">No data yet.</p>
        )}
      </Card>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card title="By category">{shareList(r.by_category, false)}</Card>
        <Card title="By vehicle">{shareList(r.by_vehicle, true)}</Card>
      </div>
    </div>
  );
}
