import { Link } from "react-router-dom";
import type { Vehicle } from "../lib/api";

/** A vehicle summary card as shown on the Vehicles and Archive lists. */
export default function VehicleCard({ v }: { v: Vehicle }) {
  return (
    <Link
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
  );
}
