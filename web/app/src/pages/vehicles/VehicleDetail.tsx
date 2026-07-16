import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  api,
  type Attachment,
  type Meta,
  type ReferenceItem,
  type Reminder,
  type ServiceRecord,
  type Vehicle,
} from "../../lib/api";
import { formatDate, miles, money } from "../../lib/format";
import StatusBadge from "../../components/StatusBadge";
import { Card } from "../../components/form";
import ReferenceEditor from "./ReferenceEditor";

interface Detail {
  vehicle: Vehicle;
  services: ServiceRecord[];
  reminders: Reminder[];
  photos: Attachment[];
  documents: Attachment[];
  reference: ReferenceItem[];
}

type Tab = "service" | "reminders" | "reference" | "files";

export default function VehicleDetail() {
  const { id } = useParams();
  const nav = useNavigate();
  const qc = useQueryClient();
  const [tab, setTab] = useState<Tab>("service");

  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });
  const q = useQuery({
    queryKey: ["vehicle", id],
    queryFn: () => api.get<Detail>(`/vehicles/${id}`),
  });

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["vehicle", id] });
    qc.invalidateQueries({ queryKey: ["vehicles"] });
    qc.invalidateQueries({ queryKey: ["dashboard"] });
    qc.invalidateQueries({ queryKey: ["reminders"] });
  };

  const del = useMutation({
    mutationFn: () => api.delete(`/vehicles/${id}`),
    onSuccess: () => {
      qc.invalidateQueries();
      nav("/vehicles");
    },
  });

  if (q.isLoading) return <p className="text-muted">Loading…</p>;
  if (q.isError || !q.data) return <p className="text-danger">Vehicle not found.</p>;

  const d = q.data;
  const v = d.vehicle;
  const currency = meta.data?.currency ?? "$";
  const unit = meta.data?.distance_unit ?? "mi";

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: "service", label: "Service", count: d.services.length },
    { key: "reminders", label: "Reminders", count: d.reminders.length },
    { key: "reference", label: "Reference", count: d.reference.length },
    { key: "files", label: "Photos & docs", count: d.photos.length + d.documents.length },
  ];

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">{v.name}</h1>
          <p className="text-muted">
            {[v.year, v.make, v.model].filter(Boolean).join(" ")}
            {v.color && ` · ${v.color}`}
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/vehicles/${v.id}/edit`} className="flex min-h-11 items-center rounded-lg border border-border px-4 hover:border-muted">
            Edit
          </Link>
          <button
            onClick={() => {
              if (confirm("Delete this vehicle and all its records?")) del.mutate();
            }}
            className="min-h-11 rounded-lg border border-border px-4 text-danger hover:bg-danger/10"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Photo + facts */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-[320px_1fr]">
        <div className="overflow-hidden rounded-xl border border-border bg-surface">
          {v.photo_id ? (
            <img src={`/api/v1/files/${v.photo_id}`} alt={v.name} className="aspect-[16/10] w-full object-cover" />
          ) : (
            <div className="flex aspect-[16/10] items-center justify-center bg-surface-2 text-6xl">🚗</div>
          )}
        </div>
        <Card>
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
            <dt className="text-muted">VIN</dt><dd className="break-all">{v.vin || "—"}</dd>
            <dt className="text-muted">Plate</dt><dd>{v.license_plate || "—"}</dd>
            <dt className="text-muted">Odometer</dt><dd>{miles(unit, v.odometer)}</dd>
            <dt className="text-muted">Purchased</dt><dd>{v.purchase_date ? formatDate(v.purchase_date) : "—"}</dd>
            <dt className="text-muted">Total spent</dt><dd>{money(currency, v.total_cost)}</dd>
          </dl>
          {v.notes && <p className="mt-3 whitespace-pre-wrap text-sm">{v.notes}</p>}
        </Card>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 overflow-x-auto border-b border-border">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`min-h-11 shrink-0 border-b-2 px-3.5 text-sm ${
              tab === t.key
                ? "border-primary font-semibold text-text"
                : "border-transparent text-muted hover:text-text"
            }`}
          >
            {t.label}
            {t.count !== undefined && t.count > 0 && <span className="ml-1.5 text-xs text-muted">{t.count}</span>}
          </button>
        ))}
      </div>

      {tab === "service" && <ServiceTab d={d} currency={currency} unit={unit} refresh={refresh} />}
      {tab === "reminders" && <RemindersTab d={d} refresh={refresh} />}
      {tab === "reference" && <ReferenceEditor vehicleId={v.id} items={d.reference} refresh={refresh} />}
      {tab === "files" && <FilesTab d={d} refresh={refresh} />}
    </div>
  );
}

// ---- Service log ----

function ServiceTab({
  d, currency, unit, refresh,
}: { d: Detail; currency: string; unit: string; refresh: () => void }) {
  const v = d.vehicle;
  const delSvc = useMutation({
    mutationFn: (sid: number) => api.delete(`/services/${sid}`),
    onSuccess: refresh,
  });

  return (
    <Card
      title="Service log"
      actions={
        <div className="flex gap-2">
          {d.services.length > 0 && (
            <a href={`/api/v1/vehicles/${v.id}/export.csv`} download className="flex min-h-9 items-center rounded-lg border border-border px-3 text-sm hover:border-muted">
              ⬇ CSV
            </a>
          )}
          <Link to={`/vehicles/${v.id}/services/new`} className="flex min-h-9 items-center rounded-lg bg-primary px-3 text-sm font-semibold text-white hover:bg-primary-strong">
            + Add
          </Link>
        </div>
      }
    >
      {d.services.length === 0 ? (
        <p className="py-6 text-center text-muted">No service records yet.</p>
      ) : (
        <ul className="divide-y divide-border">
          {d.services.map((sr) => (
            <li key={sr.id} className="flex flex-wrap items-baseline gap-x-3 gap-y-1 py-3">
              <div className="min-w-0 flex-1">
                <div className="font-medium">
                  {sr.description}
                  {(sr.attachments?.length ?? 0) > 0 && " 📎"}
                </div>
                <div className="mt-0.5 text-sm text-muted">
                  {formatDate(sr.date)}
                  {sr.odometer != null && ` · ${miles(unit, sr.odometer)}`}
                  {" · "}
                  <span className="rounded-full bg-surface-2 px-2 py-0.5 text-xs">{sr.category}</span>
                  {sr.vendor && ` · ${sr.vendor}`}
                </div>
                {sr.notes && <div className="mt-0.5 text-sm text-muted">{sr.notes}</div>}
              </div>
              <div className="tabular-nums font-medium">{sr.cost > 0 ? money(currency, sr.cost) : ""}</div>
              <div className="flex shrink-0 gap-1">
                <Link to={`/services/${sr.id}/edit`} className="flex min-h-9 min-w-9 items-center justify-center rounded text-sm text-muted hover:text-text">
                  edit
                </Link>
                <button
                  onClick={() => confirm("Delete this record?") && delSvc.mutate(sr.id)}
                  className="flex min-h-9 min-w-9 items-center justify-center rounded text-sm text-muted hover:text-danger"
                >
                  ✕
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

// ---- Reminders ----

function RemindersTab({ d, refresh }: { d: Detail; refresh: () => void }) {
  const complete = useMutation({
    mutationFn: (rid: number) => api.post(`/reminders/${rid}/complete`),
    onSuccess: refresh,
  });
  const del = useMutation({
    mutationFn: (rid: number) => api.delete(`/reminders/${rid}`),
    onSuccess: refresh,
  });

  return (
    <Card
      title="Reminders"
      actions={
        <Link to={`/vehicles/${d.vehicle.id}/reminders/new`} className="flex min-h-9 items-center rounded-lg bg-primary px-3 text-sm font-semibold text-white hover:bg-primary-strong">
          + Add
        </Link>
      }
    >
      {d.reminders.length === 0 ? (
        <p className="py-6 text-center text-muted">No active reminders.</p>
      ) : (
        <ul className="divide-y divide-border">
          {d.reminders.map((r) => (
            <li key={r.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 py-3">
              <StatusBadge status={r.status} text={r.status_text} />
              <div className="min-w-0 flex-1">
                <span className="font-medium">{r.title}</span>
                {r.notify && <span title="Emails when due"> 📧</span>}
                {r.notes && <div className="text-sm text-muted">{r.notes}</div>}
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
  );
}

// ---- Photos & documents ----

function FilesTab({ d, refresh }: { d: Detail; refresh: () => void }) {
  const v = d.vehicle;
  const upload = useMutation({
    mutationFn: ({ field, file, path }: { field: string; file: File; path: string }) =>
      api.upload(path, field, file),
    onSuccess: refresh,
  });
  const delAtt = useMutation({
    mutationFn: (aid: number) => api.delete(`/attachments/${aid}`),
    onSuccess: refresh,
  });
  const setPrimary = useMutation({
    mutationFn: (aid: number) => api.post(`/vehicles/${v.id}/photo/${aid}/primary`),
    onSuccess: refresh,
  });

  const pick = (accept: string, capture: boolean, cb: (f: File) => void) => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = accept;
    if (capture) input.setAttribute("capture", "environment");
    input.onchange = () => input.files?.[0] && cb(input.files[0]);
    input.click();
  };

  return (
    <div className="space-y-4">
      <Card
        title="Photos"
        actions={
          <button
            onClick={() => pick("image/*", true, (f) => upload.mutate({ field: "photo", file: f, path: `/vehicles/${v.id}/photos` }))}
            className="flex min-h-9 items-center rounded-lg bg-primary px-3 text-sm font-semibold text-white hover:bg-primary-strong"
          >
            + Photo
          </button>
        }
      >
        {d.photos.length === 0 ? (
          <p className="py-4 text-center text-muted">No photos yet.</p>
        ) : (
          <div className="flex snap-x gap-3 overflow-x-auto pb-1">
            {d.photos.map((p) => (
              <div key={p.id} className="relative shrink-0 snap-start">
                <img src={`/api/v1/files/${p.id}`} alt={p.original_name} className="h-28 w-40 rounded-lg object-cover" />
                <div className="absolute right-1 top-1 flex gap-1">
                  <button
                    onClick={() => setPrimary.mutate(p.id)}
                    title="Set as main photo"
                    className={`flex h-8 w-8 items-center justify-center rounded-full bg-black/60 text-sm ${v.photo_id === p.id ? "text-soon" : "text-white"}`}
                  >
                    ★
                  </button>
                  <button
                    onClick={() => confirm("Delete this photo?") && delAtt.mutate(p.id)}
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-black/60 text-sm text-white hover:text-danger"
                  >
                    ✕
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card
        title="Documents"
        actions={
          <button
            onClick={() => pick("*/*", false, (f) => upload.mutate({ field: "document", file: f, path: `/vehicles/${v.id}/documents` }))}
            className="flex min-h-9 items-center rounded-lg bg-primary px-3 text-sm font-semibold text-white hover:bg-primary-strong"
          >
            + Document
          </button>
        }
      >
        {d.documents.length === 0 ? (
          <p className="py-4 text-center text-muted">No documents yet.</p>
        ) : (
          <ul className="divide-y divide-border">
            {d.documents.map((doc) => (
              <li key={doc.id} className="flex items-center gap-2 py-2.5">
                <a href={`/api/v1/files/${doc.id}`} target="_blank" rel="noreferrer" className="min-w-0 flex-1 truncate text-primary hover:underline">
                  📄 {doc.original_name}
                </a>
                <button
                  onClick={() => confirm("Delete this document?") && delAtt.mutate(doc.id)}
                  className="flex min-h-9 min-w-9 items-center justify-center text-sm text-muted hover:text-danger"
                >
                  ✕
                </button>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
