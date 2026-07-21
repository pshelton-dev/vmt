import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api, type Vehicle } from "../../lib/api";
import VehicleCard from "../../components/VehicleCard";

export default function Archive() {
  const q = useQuery({
    queryKey: ["vehicles", "archived"],
    queryFn: () => api.get<Vehicle[]>("/vehicles/archived"),
  });

  if (q.isLoading) return <p className="text-muted">Loading…</p>;
  if (q.isError || !q.data) return <p className="text-danger">Could not load archived vehicles.</p>;

  return (
    <div className="space-y-5">
      <div>
        <Link to="/vehicles" className="text-sm text-muted hover:text-text">
          ← Vehicles
        </Link>
        <h1 className="mt-1 text-2xl font-bold">Archived vehicles</h1>
        <p className="text-muted">
          Vehicles you no longer own. Their records are kept and still count in Reports,
          but they’re left out of the fleet list, dashboard totals and reminders.
        </p>
      </div>

      {q.data.length === 0 ? (
        <p className="rounded-xl border border-border bg-surface p-10 text-center text-muted">
          Nothing archived. Archive a vehicle from its page when you sell or retire it.
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {q.data.map((v) => (
            <VehicleCard key={v.id} v={v} />
          ))}
        </div>
      )}
    </div>
  );
}
