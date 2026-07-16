import { useEffect, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { api, ApiError, type Meta, type ServiceRecord, type Vehicle } from "../../lib/api";
import { Card, Field, PrimaryButton, SecondaryButton, Select, TextArea, TextInput } from "../../components/form";

interface FormState {
  date: string;
  odometer: string;
  category: string;
  description: string;
  vendor: string;
  cost: string;
  notes: string;
}

/** Add a service (route param vid) or edit one (route param sid). */
export default function ServiceForm() {
  const { vid, sid } = useParams();
  const isNew = !sid;
  const nav = useNavigate();
  const qc = useQueryClient();

  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });
  const existing = useQuery({
    queryKey: ["service", sid],
    enabled: !isNew,
    queryFn: () => api.get<ServiceRecord>(`/services/${sid}`),
  });
  const vehicleId = isNew ? Number(vid) : existing.data?.vehicle_id;
  const vehicle = useQuery({
    queryKey: ["vehicle", String(vehicleId)],
    enabled: !!vehicleId,
    queryFn: () => api.get<{ vehicle: Vehicle }>(`/vehicles/${vehicleId}`),
  });

  const [form, setForm] = useState<FormState>({
    date: new Date().toISOString().slice(0, 10),
    odometer: "",
    category: "Oil Change",
    description: "",
    vendor: "",
    cost: "",
    notes: "",
  });
  const [receipt, setReceipt] = useState<File | null>(null);
  const [error, setError] = useState("");

  // Prefill: edit loads the record; add prefills odometer from the vehicle.
  useEffect(() => {
    if (existing.data) {
      const sr = existing.data;
      setForm({
        date: sr.date,
        odometer: sr.odometer != null ? String(sr.odometer) : "",
        category: sr.category,
        description: sr.description,
        vendor: sr.vendor,
        cost: sr.cost ? String(sr.cost) : "",
        notes: sr.notes,
      });
    }
  }, [existing.data]);
  useEffect(() => {
    if (isNew && vehicle.data && vehicle.data.vehicle.odometer > 0) {
      setForm((f) => (f.odometer ? f : { ...f, odometer: String(vehicle.data.vehicle.odometer) }));
    }
  }, [isNew, vehicle.data]);

  const set = (k: keyof FormState) => (e: { target: { value: string } }) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const save = useMutation({
    mutationFn: async () => {
      const body = {
        date: form.date,
        odometer: form.odometer ? Number(form.odometer) : null,
        category: form.category,
        description: form.description,
        vendor: form.vendor,
        cost: form.cost ? Number(form.cost) : 0,
        notes: form.notes,
      };
      const sr = isNew
        ? await api.post<ServiceRecord>(`/vehicles/${vid}/services`, body)
        : await api.put<ServiceRecord>(`/services/${sid}`, body);
      if (receipt) await api.upload(`/services/${sr.id}/attachments`, "document", receipt);
      return sr;
    },
    onSuccess: (sr) => {
      qc.invalidateQueries();
      nav(`/vehicles/${sr.vehicle_id}`);
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Save failed."),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError("");
    save.mutate();
  };

  const categories = meta.data?.service_categories ?? ["Other"];
  const vehicleName = vehicle.data?.vehicle.name;

  return (
    <form onSubmit={submit} className="mx-auto max-w-2xl space-y-5">
      <h1 className="text-2xl font-bold">
        {isNew ? "Add service" : "Edit service"}
        {vehicleName && <span className="text-muted"> · {vehicleName}</span>}
      </h1>
      <Card>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Date *">
            <TextInput type="date" value={form.date} onChange={set("date")} required />
          </Field>
          <Field label="Odometer">
            <TextInput type="number" min={0} value={form.odometer} onChange={set("odometer")} inputMode="numeric" />
          </Field>
          <Field label="Category">
            <Select value={form.category} onChange={set("category")}>
              {categories.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </Select>
          </Field>
          <Field label="Cost">
            <TextInput type="number" min={0} step="0.01" value={form.cost} onChange={set("cost")} inputMode="decimal" />
          </Field>
        </div>
        <div className="mt-4 space-y-4">
          <Field label="Description *">
            <TextInput value={form.description} onChange={set("description")} required placeholder="Oil & filter change" />
          </Field>
          <Field label="Vendor / shop">
            <TextInput value={form.vendor} onChange={set("vendor")} />
          </Field>
          <Field label="Notes">
            <TextArea rows={3} value={form.notes} onChange={set("notes")} />
          </Field>
          <Field label="Attach receipt / document">
            <input
              type="file"
              capture="environment"
              onChange={(e) => setReceipt(e.target.files?.[0] ?? null)}
              className="mt-1 block w-full text-sm text-muted file:mr-3 file:rounded-lg file:border file:border-border file:bg-surface-2 file:px-3 file:py-2 file:text-text"
            />
          </Field>
        </div>
      </Card>
      {error && <p className="text-danger">{error}</p>}
      <div className="flex gap-3">
        <PrimaryButton disabled={save.isPending}>{save.isPending ? "Saving…" : "Save"}</PrimaryButton>
        <SecondaryButton onClick={() => nav(vehicleId ? `/vehicles/${vehicleId}` : "/vehicles")}>
          Cancel
        </SecondaryButton>
      </div>
    </form>
  );
}
