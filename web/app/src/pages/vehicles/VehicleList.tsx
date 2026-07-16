import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api, type Vehicle } from "../../lib/api";

export default function VehicleList() {
  const q = useQuery({ queryKey: ["vehicles"], queryFn: () => api.get<Vehicle[]>("/vehicles") });

  if (q.isLoading) return <p className="text-muted">Loading…</p>;
  if (q.isError || !q.data) return <p className="text-danger">Could not load vehicles.</p>;

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Vehicles</h1>
        <Link
          to="/vehicles/new"
          className="flex min-h-11 items-center rounded-lg bg-primary px-4 font-semibold text-white hover:bg-primary-strong"
        >
          + Add vehicle
        </Link>
      </div>

      {q.data.length === 0 ? (
        <p className="rounded-xl border border-border bg-surface p-10 text-center text-muted">
          No vehicles yet — add your first one.
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {q.data.map((v) => (
            <Link
              key={v.id}
              to={`/vehicles/${v.id}`}
              className="overflow-hidden rounded-xl border border-border bg-surface transition-colors hover:border-primary"
            >
              <div className="flex aspect-[16/10] items-center justify-center bg-surface-2 text-5xl">
                {v.photo_id ? (
                  <img
                    src={`/api/v1/files/${v.photo_id}`}
                    alt={v.name}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <span>🚗</span>
                )}
              </div>
              <div className="p-3">
                <div className="flex items-center gap-2">
                  <span className="truncate font-semibold">{v.name}</span>
                  {v.due_reminders > 0 && (
                    <span className="ml-auto shrink-0 rounded-full bg-due px-2 py-0.5 text-xs font-bold text-white">
                      {v.due_reminders} due
                    </span>
                  )}
                </div>
                <div className="mt-0.5 text-sm text-muted">
                  {[v.year, v.make, v.model].filter(Boolean).join(" ") || "—"} ·{" "}
                  {v.service_count} records
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
