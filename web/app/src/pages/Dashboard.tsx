import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api, type Dashboard as DashboardData, type Meta } from "../lib/api";
import { money, formatDate } from "../lib/format";
import StatusBadge from "../components/StatusBadge";

export default function Dashboard() {
  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });
  const dash = useQuery({
    queryKey: ["dashboard"],
    queryFn: () => api.get<DashboardData>("/dashboard"),
  });

  if (dash.isLoading) return <p className="text-muted">Loading…</p>;
  if (dash.isError || !dash.data)
    return <p className="text-danger">Could not load the dashboard.</p>;

  const d = dash.data;
  const currency = meta.data?.currency ?? "$";

  const stats = [
    { label: "Vehicles", value: String(d.vehicles.length) },
    { label: "Service records", value: String(d.service_count) },
    { label: "Total spent", value: money(currency, d.total_cost) },
    {
      label: "Needs attention",
      value: String(d.due_reminders.length),
      warn: d.due_reminders.length > 0,
    },
  ];

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {stats.map((s) => (
          <div key={s.label} className="rounded-xl border border-border bg-surface p-4">
            <div className={`text-2xl font-bold ${s.warn ? "text-due" : ""}`}>{s.value}</div>
            <div className="text-sm text-muted">{s.label}</div>
          </div>
        ))}
      </div>

      {d.due_reminders.length > 0 && (
        <section className="rounded-xl border border-border bg-surface p-4">
          <h2 className="mb-3 font-semibold">Needs attention</h2>
          <ul className="divide-y divide-border">
            {d.due_reminders.map((r) => (
              <li key={r.id} className="flex items-center gap-3 py-2.5">
                <StatusBadge status={r.status} text={r.status_text} />
                <div className="min-w-0">
                  <div className="truncate font-medium">{r.title}</div>
                  <div className="text-sm text-muted">{r.vehicle_name}</div>
                </div>
                <Link
                  to={`/vehicles/${r.vehicle_id}`}
                  className="ml-auto shrink-0 text-sm text-primary"
                >
                  View →
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="rounded-xl border border-border bg-surface p-4">
        <h2 className="mb-3 font-semibold">Recent service</h2>
        {d.recent_services.length === 0 ? (
          <p className="py-4 text-center text-muted">No service records yet.</p>
        ) : (
          <ul className="divide-y divide-border">
            {d.recent_services.map((sr) => (
              <li key={sr.id} className="flex items-baseline gap-3 py-2.5">
                <div className="min-w-0">
                  <div className="truncate font-medium">{sr.description}</div>
                  <div className="text-sm text-muted">
                    {sr.vehicle_name} · {formatDate(sr.date)}
                  </div>
                </div>
                {sr.cost > 0 && (
                  <div className="ml-auto shrink-0 tabular-nums">
                    {money(currency, sr.cost)}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
