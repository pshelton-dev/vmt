import { useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiError, type Meta, type Settings as SettingsData } from "../lib/api";
import { applyThemePref, getThemePref, type ThemePref } from "../lib/theme";
import { Card, Field, PrimaryButton, Select, TextInput, inputCls } from "../components/form";

const themeOptions: { value: ThemePref; label: string }[] = [
  { value: "auto", label: "Auto (device)" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

function useFlash(): [string, (msg: string, isError?: boolean) => void, boolean] {
  const [msg, setMsg] = useState("");
  const [isError, setIsError] = useState(false);
  const flash = (m: string, err = false) => {
    setMsg(m);
    setIsError(err);
  };
  return [msg, flash, isError];
}

export default function SettingsPage() {
  const qc = useQueryClient();
  const settings = useQuery({ queryKey: ["settings"], queryFn: () => api.get<SettingsData>("/settings") });
  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });

  if (settings.isLoading) return <p className="text-muted">Loading…</p>;
  if (settings.isError || !settings.data) return <p className="text-danger">Could not load settings.</p>;

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["settings"] });
    qc.invalidateQueries({ queryKey: ["meta"] });
  };

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-bold">Settings</h1>
      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
        <AppearanceCard />
        <PreferencesCard s={settings.data} presets={meta.data?.date_presets ?? []} refresh={refresh} />
        <PasswordCard />
        <NotificationsCard s={settings.data} refresh={refresh} />
        <ImportCard />
        <BackupCard />
      </div>
    </div>
  );
}

// ---- appearance (client-side only) ----

function AppearanceCard() {
  const [theme, setTheme] = useState<ThemePref>(getThemePref());
  return (
    <Card title="Appearance">
      <div className="flex gap-2">
        {themeOptions.map((o) => (
          <button
            key={o.value}
            onClick={() => {
              applyThemePref(o.value);
              setTheme(o.value);
            }}
            className={`min-h-11 flex-1 rounded-lg border px-3 text-sm ${
              theme === o.value
                ? "border-primary bg-primary/10 font-semibold text-primary"
                : "border-border text-muted hover:text-text"
            }`}
          >
            {o.label}
          </button>
        ))}
      </div>
    </Card>
  );
}

// ---- display preferences ----

function PreferencesCard({
  s, presets, refresh,
}: { s: SettingsData; presets: { layout: string; example: string }[]; refresh: () => void }) {
  const [currency, setCurrency] = useState(s.currency);
  const [unit, setUnit] = useState(s.distance_unit);
  const [format, setFormat] = useState(s.date_format);
  const [msg, flash, isErr] = useFlash();

  const save = useMutation({
    mutationFn: () =>
      api.put("/settings", { currency, distance_unit: unit, date_format: format }),
    onSuccess: () => {
      flash("Preferences saved.");
      refresh();
    },
    onError: (e) => flash(e instanceof ApiError ? e.message : "Save failed.", true),
  });

  return (
    <Card title="Display preferences">
      <form
        onSubmit={(e: FormEvent) => {
          e.preventDefault();
          save.mutate();
        }}
        className="space-y-4"
      >
        <div className="grid grid-cols-2 gap-4">
          <Field label="Currency symbol">
            <TextInput value={currency} onChange={(e) => setCurrency(e.target.value)} maxLength={3} />
          </Field>
          <Field label="Distance unit">
            <TextInput value={unit} onChange={(e) => setUnit(e.target.value)} maxLength={8} />
          </Field>
        </div>
        <Field label="Date format (classic UI)">
          <Select value={format} onChange={(e) => setFormat(e.target.value)}>
            {presets.map((p) => (
              <option key={p.layout} value={p.layout}>{p.example}</option>
            ))}
          </Select>
        </Field>
        {msg && <p className={`text-sm ${isErr ? "text-danger" : "text-ok"}`}>{msg}</p>}
        <PrimaryButton disabled={save.isPending}>Save preferences</PrimaryButton>
      </form>
    </Card>
  );
}

// ---- password ----

function PasswordCard() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirmPw, setConfirmPw] = useState("");
  const [msg, flash, isErr] = useFlash();

  const save = useMutation({
    mutationFn: () => api.post("/settings/password", { current, new: next }),
    onSuccess: () => {
      flash("Password changed.");
      setCurrent(""); setNext(""); setConfirmPw("");
    },
    onError: (e) => flash(e instanceof ApiError ? e.message : "Change failed.", true),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (next !== confirmPw) {
      flash("New passwords do not match.", true);
      return;
    }
    save.mutate();
  };

  return (
    <Card title="Change password">
      <form onSubmit={submit} className="space-y-4">
        <Field label="Current password">
          <TextInput type="password" value={current} onChange={(e) => setCurrent(e.target.value)} required autoComplete="current-password" />
        </Field>
        <Field label="New password">
          <TextInput type="password" value={next} onChange={(e) => setNext(e.target.value)} minLength={6} required autoComplete="new-password" />
        </Field>
        <Field label="Confirm new password">
          <TextInput type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} minLength={6} required autoComplete="new-password" />
        </Field>
        {msg && <p className={`text-sm ${isErr ? "text-danger" : "text-ok"}`}>{msg}</p>}
        <PrimaryButton disabled={save.isPending}>Update password</PrimaryButton>
      </form>
    </Card>
  );
}

// ---- notifications ----

function NotificationsCard({ s, refresh }: { s: SettingsData; refresh: () => void }) {
  const [email, setEmail] = useState(s.notify_email);
  const [enabled, setEnabled] = useState(s.notify_enabled);
  const [msg, flash, isErr] = useFlash();

  const save = useMutation({
    mutationFn: () => api.put("/settings", { notify_email: email, notify_enabled: enabled }),
    onSuccess: () => {
      flash("Notification settings saved.");
      refresh();
    },
    onError: (e) => flash(e instanceof ApiError ? e.message : "Save failed.", true),
  });
  const test = useMutation({
    mutationFn: async () => {
      await api.put("/settings", { notify_email: email, notify_enabled: enabled });
      return api.post<{ sent_to: string }>("/settings/test-email");
    },
    onSuccess: (r) => flash(`Test email sent to ${r.sent_to}.`),
    onError: (e) => flash(e instanceof ApiError ? e.message : "Test failed.", true),
  });

  return (
    <Card title="Email notifications">
      <p className="mb-3 text-sm text-muted">
        Get an email when a reminder you've opted into becomes due (per-reminder
        “Email me when this is due” checkbox).
      </p>
      <p className="mb-3 text-sm">
        {s.smtp_configured ? (
          <>📧 SMTP is configured (sending as <code className="rounded bg-surface-2 px-1">{s.smtp_from}</code>).</>
        ) : (
          <>⚠ SMTP is not configured — set the <code className="rounded bg-surface-2 px-1">VMT_SMTP_*</code> environment variables and restart.</>
        )}
      </p>
      <form
        onSubmit={(e: FormEvent) => {
          e.preventDefault();
          save.mutate();
        }}
        className="space-y-4"
      >
        <Field label="Notification recipient">
          <TextInput type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" />
        </Field>
        <label className="flex min-h-11 items-center gap-2.5 text-sm">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="h-5 w-5 accent-[var(--primary)]"
          />
          Enable email notifications
        </label>
        {msg && <p className={`text-sm ${isErr ? "text-danger" : "text-ok"}`}>{msg}</p>}
        <div className="flex flex-wrap gap-3">
          <PrimaryButton disabled={save.isPending}>Save notifications</PrimaryButton>
          <button
            type="button"
            onClick={() => test.mutate()}
            disabled={test.isPending || !s.smtp_configured}
            className="min-h-11 rounded-lg border border-border px-5 hover:border-muted disabled:opacity-50"
          >
            {test.isPending ? "Sending…" : "Send test email"}
          </button>
        </div>
      </form>
    </Card>
  );
}

// ---- CSV import with preview ----

interface PreviewRow {
  line: number;
  ok: boolean;
  reason?: string;
  vehicle?: string;
  new_vehicle?: boolean;
  date?: string;
  category?: string;
  description?: string;
  cost?: number;
}

interface Preview {
  rows: PreviewRow[];
  new_vehicles: string[];
  would_import: number;
  skipped: number;
  csv_data: string;
}

function ImportCard() {
  const qc = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [msg, flash, isErr] = useFlash();

  const doPreview = useMutation({
    mutationFn: (file: File) => {
      const form = new FormData();
      form.append("csv", file);
      return fetch("/api/v1/import/preview", { method: "POST", body: form }).then(async (r) => {
        const body = await r.json();
        if (!r.ok) throw new ApiError(r.status, body?.error ?? "preview failed");
        return body as Preview;
      });
    },
    onSuccess: (p) => {
      setPreview(p);
      flash("");
    },
    onError: (e) => flash(e instanceof ApiError ? e.message : "Preview failed.", true),
  });

  const doImport = useMutation({
    mutationFn: () =>
      api.post<{ summary: string }>("/import", { csv_data: preview!.csv_data }),
    onSuccess: (r) => {
      setPreview(null);
      if (fileRef.current) fileRef.current.value = "";
      flash(r.summary);
      qc.invalidateQueries();
    },
    onError: (e) => flash(e instanceof ApiError ? e.message : "Import failed.", true),
  });

  return (
    <Card title="Import service records (CSV)">
      <p className="mb-3 text-sm text-muted">
        Columns: <code className="rounded bg-surface-2 px-1">Vehicle, Date, Odometer, Category, Description, Vendor, Cost, Notes</code>{" "}
        — the same format Export produces. Vehicles are matched by name and created
        automatically. Importing adds records; it never deletes anything.
      </p>
      <input
        ref={fileRef}
        type="file"
        accept=".csv,text/csv"
        onChange={(e) => {
          const f = e.target.files?.[0];
          setPreview(null);
          if (f) doPreview.mutate(f);
        }}
        className="mb-3 block w-full text-sm text-muted file:mr-3 file:rounded-lg file:border file:border-border file:bg-surface-2 file:px-3 file:py-2 file:text-text"
      />

      {doPreview.isPending && <p className="text-sm text-muted">Analyzing…</p>}

      {preview && (
        <div className="space-y-3">
          <p className="text-sm">
            Would import <strong>{preview.would_import}</strong> record(s)
            {preview.new_vehicles.length > 0 && (
              <>, creating vehicle(s): <strong>{preview.new_vehicles.join(", ")}</strong></>
            )}
            {preview.skipped > 0 && (
              <span className="text-due"> · {preview.skipped} row(s) will be skipped</span>
            )}
          </p>
          <div className="max-h-56 overflow-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-surface-2 text-left text-muted">
                <tr>
                  <th className="p-2">Line</th>
                  <th className="p-2">Vehicle</th>
                  <th className="p-2">Date</th>
                  <th className="p-2">Description</th>
                  <th className="p-2">Status</th>
                </tr>
              </thead>
              <tbody>
                {preview.rows.map((row) => (
                  <tr key={row.line} className="border-t border-border">
                    <td className="p-2 text-muted">{row.line}</td>
                    <td className="p-2">
                      {row.vehicle}
                      {row.new_vehicle && <span className="text-primary"> (new)</span>}
                    </td>
                    <td className="p-2">{row.date}</td>
                    <td className="p-2">{row.description}</td>
                    <td className="p-2">
                      {row.ok ? <span className="text-ok">ok</span> : <span className="text-danger">{row.reason}</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => doImport.mutate()}
              disabled={doImport.isPending || preview.would_import === 0}
              className="min-h-11 rounded-lg bg-primary px-5 font-semibold text-white hover:bg-primary-strong disabled:opacity-50"
            >
              {doImport.isPending ? "Importing…" : `Import ${preview.would_import} record(s)`}
            </button>
            <button
              onClick={() => {
                setPreview(null);
                if (fileRef.current) fileRef.current.value = "";
              }}
              className="min-h-11 rounded-lg border border-border px-5 hover:border-muted"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {msg && <p className={`mt-2 text-sm ${isErr ? "text-danger" : "text-ok"}`}>{msg}</p>}
    </Card>
  );
}

// ---- backup & restore ----

function BackupCard() {
  const [msg, flash, isErr] = useFlash();
  const [restoring, setRestoring] = useState(false);

  const restore = async (file: File) => {
    if (!confirm("Restore will REPLACE all current data with this backup, then restart the app. Continue?")) return;
    setRestoring(true);
    flash("");
    try {
      const form = new FormData();
      form.append("backup", file);
      const res = await fetch("/api/v1/restore", { method: "POST", body: form });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.error ?? `HTTP ${res.status}`);
      }
      flash("Backup restored — the app is restarting. This page will reload shortly.");
      setTimeout(() => window.location.reload(), 5000);
    } catch (e) {
      flash(e instanceof Error ? e.message : "Restore failed.", true);
      setRestoring(false);
    }
  };

  return (
    <Card title="Backup & restore">
      <p className="mb-3 text-sm text-muted">
        Download a complete archive of your database and uploaded files, or restore
        from a previous backup.
      </p>
      <a
        href="/api/v1/backup"
        download
        className="inline-flex min-h-11 items-center rounded-lg border border-border px-4 text-sm hover:border-muted"
      >
        ⬇ Download backup (.tar.gz)
      </a>
      <hr className="my-4 border-border" />
      <Field label="Restore from backup">
        <input
          type="file"
          accept=".gz,.tgz,application/gzip"
          disabled={restoring}
          onChange={(e) => e.target.files?.[0] && restore(e.target.files[0])}
          className={inputCls + " file:mr-3 file:rounded file:border-0 file:bg-surface-2 file:px-3 file:py-1.5 file:text-text"}
        />
      </Field>
      <p className="mt-2 text-sm text-muted">
        ⚠ This overwrites everything. Current data is kept as{" "}
        <code className="rounded bg-surface-2 px-1">vmt.db.prerestore</code> in the data
        directory just in case.
      </p>
      {msg && <p className={`mt-2 text-sm ${isErr ? "text-danger" : "text-ok"}`}>{msg}</p>}
    </Card>
  );
}
