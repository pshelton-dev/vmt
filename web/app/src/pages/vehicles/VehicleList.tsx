import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api, type Vehicle } from "../../lib/api";
import VehicleCard from "../../components/VehicleCard";

export default function VehicleList() {
  const q = useQuery({ queryKey: ["vehicles"], queryFn: () => api.get<Vehicle[]>("/vehicles") });
  const archived = useQuery({
    queryKey: ["vehicles", "archived"],
    queryFn: () => api.get<Vehicle[]>("/vehicles/archived"),
  });

  if (q.isLoading) return <p className="text-muted">Loading…</p>;
  if (q.isError || !q.data) return <p className="text-danger">Could not load vehicles.</p>;

  const archivedCount = archived.data?.length ?? 0;

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
            <VehicleCard key={v.id} v={v} />
          ))}
        </div>
      )}

      {archivedCount > 0 && (
        <Link
          to="/vehicles/archive"
          className="flex min-h-11 items-center gap-2 text-sm text-muted hover:text-text"
        >
          🗄️ Archived vehicles
          <span className="rounded-full bg-surface-2 px-2 py-0.5 text-xs font-semibold">
            {archivedCount}
          </span>
        </Link>
      )}
    </div>
  );
}
