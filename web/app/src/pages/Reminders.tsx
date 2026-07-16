import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api, type Reminder } from "../lib/api";
import StatusBadge from "../components/StatusBadge";
import { Card } from "../components/form";

export default function Reminders() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["reminders"], queryFn: () => api.get<Reminder[]>("/reminders") });

  const refresh = () => qc.invalidateQueries();
  const complete = useMutation({
    mutationFn: (rid: number) => api.post(`/reminders/${rid}/complete`),
    onSuccess: refresh,
  });
  const del = useMutation({
    mutationFn: (rid: number) => api.delete(`/reminders/${rid}`),
    onSuccess: refresh,
  });

  if (q.isLoading) return <p className="text-muted">Loading…</p>;
  if (q.isError || !q.data) return <p className="text-danger">Could not load reminders.</p>;

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-bold">Reminders</h1>
      <Card>
        {q.data.length === 0 ? (
          <p className="py-8 text-center text-muted">
            No active reminders. Add them from a vehicle's page.
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {q.data.map((r) => (
              <li key={r.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 py-3">
                <StatusBadge status={r.status} text={r.status_text} />
                <div className="min-w-0 flex-1">
                  <span className="font-medium">{r.title}</span>
                  {r.notify && <span title="Emails when due"> 📧</span>}
                  <div className="text-sm text-muted">
                    <Link to={`/vehicles/${r.vehicle_id}`} className="text-primary hover:underline">
                      {r.vehicle_name}
                    </Link>
                    {r.notes && ` — ${r.notes}`}
                  </div>
                </div>
                <div className="flex shrink-0 gap-1">
                  <button
                    onClick={() => complete.mutate(r.id)}
                    className="flex min-h-9 items-center rounded border border-border px-2.5 text-sm text-muted hover:text-text"
                  >
                    done
                  </button>
                  <Link to={`/reminders/${r.id}/edit`} className="flex min-h-9 items-center px-2 text-sm text-muted hover:text-text">
                    edit
                  </Link>
                  <button
                    onClick={() => confirm("Delete this reminder?") && del.mutate(r.id)}
                    className="flex min-h-9 min-w-9 items-center justify-center text-sm text-muted hover:text-danger"
                  >
                    ✕
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
