import { useEffect, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { api, ApiError, type Reminder, type Settings } from "../../lib/api";
import { Card, Field, PrimaryButton, SecondaryButton, TextArea, TextInput } from "../../components/form";

interface FormState {
  title: string;
  due_date: string;
  due_odometer: string;
  interval_months: string;
  interval_miles: string;
  notes: string;
  notify: boolean;
}

const empty: FormState = {
  title: "", due_date: "", due_odometer: "",
  interval_months: "", interval_miles: "", notes: "", notify: false,
};

/** Add a reminder (route param vid) or edit one (route param rid). */
export default function ReminderForm() {
  const { vid, rid } = useParams();
  const isNew = !rid;
  const nav = useNavigate();
  const qc = useQueryClient();

  const settings = useQuery({ queryKey: ["settings"], queryFn: () => api.get<Settings>("/settings") });
  const existing = useQuery({
    queryKey: ["reminder", rid],
    enabled: !isNew,
    queryFn: () => api.get<Reminder>(`/reminders/${rid}`),
  });

  const [form, setForm] = useState<FormState>(empty);
  const [error, setError] = useState("");
  useEffect(() => {
    if (existing.data) {
      const r = existing.data;
      setForm({
        title: r.title,
        due_date: r.due_date ?? "",
        due_odometer: r.due_odometer != null ? String(r.due_odometer) : "",
        interval_months: r.interval_months != null ? String(r.interval_months) : "",
        interval_miles: r.interval_miles != null ? String(r.interval_miles) : "",
        notes: r.notes,
        notify: r.notify,
      });
    }
  }, [existing.data]);

  const set = (k: keyof FormState) => (e: { target: { value: string } }) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const save = useMutation({
    mutationFn: () => {
      const body = {
        title: form.title,
        due_date: form.due_date || null,
        due_odometer: form.due_odometer ? Number(form.due_odometer) : null,
        interval_months: form.interval_months ? Number(form.interval_months) : null,
        interval_miles: form.interval_miles ? Number(form.interval_miles) : null,
        notes: form.notes,
        notify: form.notify,
      };
      return isNew
        ? api.post<Reminder>(`/vehicles/${vid}/reminders`, body)
        : api.put<Reminder>(`/reminders/${rid}`, body);
    },
    onSuccess: (r) => {
      qc.invalidateQueries();
      nav(`/vehicles/${r.vehicle_id}`);
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Save failed."),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError("");
    save.mutate();
  };

  const mailReady =
    settings.data?.smtp_configured && settings.data.notify_enabled && !!settings.data.notify_email;
  const vehicleName = existing.data?.vehicle_name;

  return (
    <form onSubmit={submit} className="mx-auto max-w-2xl space-y-5">
      <h1 className="text-2xl font-bold">
        {isNew ? "Add reminder" : "Edit reminder"}
        {vehicleName && <span className="text-muted"> · {vehicleName}</span>}
      </h1>
      <Card>
        <div className="space-y-4">
          <Field label="Title *">
            <TextInput value={form.title} onChange={set("title")} required placeholder="Oil change" />
          </Field>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Due date">
              <TextInput type="date" value={form.due_date} onChange={set("due_date")} />
            </Field>
            <Field label="Due at odometer">
              <TextInput type="number" min={0} value={form.due_odometer} onChange={set("due_odometer")} inputMode="numeric" />
            </Field>
            <Field label="Repeat every (months)">
              <TextInput type="number" min={0} value={form.interval_months} onChange={set("interval_months")} inputMode="numeric" placeholder="e.g. 6" />
            </Field>
            <Field label="Repeat every (miles)">
              <TextInput type="number" min={0} value={form.interval_miles} onChange={set("interval_miles")} inputMode="numeric" placeholder="e.g. 5000" />
            </Field>
          </div>
          <p className="text-sm text-muted">
            With a repeat interval set, completing the reminder automatically schedules the next one.
          </p>
          <Field label="Notes">
            <TextArea rows={2} value={form.notes} onChange={set("notes")} />
          </Field>
          <label className="flex min-h-11 items-center gap-2.5 text-sm">
            <input
              type="checkbox"
              checked={form.notify}
              onChange={(e) => setForm((f) => ({ ...f, notify: e.target.checked }))}
              className="h-5 w-5 accent-[var(--primary)]"
            />
            Email me when this is due
            {!mailReady && (
              <span className="text-muted">(email isn’t fully configured — see Settings)</span>
            )}
          </label>
        </div>
      </Card>
      {error && <p className="text-danger">{error}</p>}
      <div className="flex gap-3">
        <PrimaryButton disabled={save.isPending}>{save.isPending ? "Saving…" : "Save"}</PrimaryButton>
        <SecondaryButton
          onClick={() => nav(isNew ? `/vehicles/${vid}` : `/vehicles/${existing.data?.vehicle_id ?? ""}`)}
        >
          Cancel
        </SecondaryButton>
      </div>
    </form>
  );
}
