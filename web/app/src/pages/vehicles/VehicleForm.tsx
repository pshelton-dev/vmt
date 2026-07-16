import { useEffect, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { api, ApiError, type Vehicle } from "../../lib/api";
import { useVehicleData, yearOptions } from "../../lib/vehicleData";
import { Card, Field, PrimaryButton, SecondaryButton, Select, TextArea, TextInput } from "../../components/form";

interface FormState {
  name: string;
  make: string;
  model: string;
  year: string;
  vin: string;
  license_plate: string;
  color: string;
  odometer: string;
  purchase_date: string;
  notes: string;
}

const empty: FormState = {
  name: "", make: "", model: "", year: "", vin: "",
  license_plate: "", color: "", odometer: "", purchase_date: "", notes: "",
};

function toState(v: Vehicle): FormState {
  return {
    name: v.name, make: v.make, model: v.model,
    year: v.year ? String(v.year) : "",
    vin: v.vin, license_plate: v.license_plate, color: v.color,
    odometer: v.odometer ? String(v.odometer) : "",
    purchase_date: v.purchase_date ?? "", notes: v.notes,
  };
}

/** Add (no :id) or edit (:id) a vehicle. */
export default function VehicleForm() {
  const { id } = useParams();
  const isNew = !id;
  const nav = useNavigate();
  const qc = useQueryClient();
  const data = useVehicleData();

  const existing = useQuery({
    queryKey: ["vehicle", id],
    enabled: !isNew,
    queryFn: () => api.get<{ vehicle: Vehicle }>(`/vehicles/${id}`),
  });

  const [form, setForm] = useState<FormState>(empty);
  const [photo, setPhoto] = useState<File | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    if (existing.data) setForm(toState(existing.data.vehicle));
  }, [existing.data]);

  const set = (k: keyof FormState) => (e: { target: { value: string } }) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const save = useMutation({
    mutationFn: async () => {
      const body = {
        name: form.name,
        make: form.make,
        model: form.model,
        year: form.year ? Number(form.year) : null,
        vin: form.vin,
        license_plate: form.license_plate,
        color: form.color,
        odometer: form.odometer ? Number(form.odometer) : 0,
        purchase_date: form.purchase_date || null,
        notes: form.notes,
      };
      const v = isNew
        ? await api.post<Vehicle>("/vehicles", body)
        : await api.put<Vehicle>(`/vehicles/${id}`, body);
      if (photo) await api.upload(`/vehicles/${v.id}/photos`, "photo", photo);
      return v;
    },
    onSuccess: (v) => {
      qc.invalidateQueries();
      nav(`/vehicles/${v.id}`);
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Save failed."),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError("");
    save.mutate();
  };

  const makes = Object.keys(data.data ?? {}).sort();
  const models = (() => {
    const key = form.make.trim().toLowerCase();
    for (const m of makes) if (m.toLowerCase() === key) return data.data![m];
    return [];
  })();

  return (
    <form onSubmit={submit} className="mx-auto max-w-2xl space-y-5">
      <h1 className="text-2xl font-bold">
        {isNew ? "Add vehicle" : `Edit ${existing.data?.vehicle.name ?? ""}`}
      </h1>
      <Card>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Name *">
            <TextInput value={form.name} onChange={set("name")} required placeholder="Daily Driver" />
          </Field>
          <Field label="Year">
            <Select value={form.year} onChange={set("year")}>
              <option value="">—</option>
              {yearOptions().map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </Select>
          </Field>
          <Field label="Make">
            <TextInput value={form.make} onChange={set("make")} list="makes-list" placeholder="Toyota" autoComplete="off" />
            <datalist id="makes-list">
              {makes.map((m) => <option key={m} value={m} />)}
            </datalist>
          </Field>
          <Field label="Model">
            <TextInput value={form.model} onChange={set("model")} list="models-list" placeholder="Tacoma" autoComplete="off" />
            <datalist id="models-list">
              {models.map((m) => <option key={m} value={m} />)}
            </datalist>
          </Field>
          <Field label="VIN">
            <TextInput value={form.vin} onChange={set("vin")} maxLength={17} placeholder="1HGCM82633A004352" />
          </Field>
          <Field label="License plate">
            <TextInput value={form.license_plate} onChange={set("license_plate")} />
          </Field>
          <Field label="Color">
            <TextInput value={form.color} onChange={set("color")} />
          </Field>
          <Field label="Current odometer">
            <TextInput type="number" min={0} value={form.odometer} onChange={set("odometer")} inputMode="numeric" />
          </Field>
          <Field label="Purchase date">
            <TextInput type="date" value={form.purchase_date} onChange={set("purchase_date")} />
          </Field>
        </div>
        <div className="mt-4 space-y-4">
          <Field label="Notes">
            <TextArea rows={3} value={form.notes} onChange={set("notes")} />
          </Field>
          <Field label={isNew ? "Photo" : "Add/replace photo"}>
            <input
              type="file"
              accept="image/*"
              capture="environment"
              onChange={(e) => setPhoto(e.target.files?.[0] ?? null)}
              className="mt-1 block w-full text-sm text-muted file:mr-3 file:rounded-lg file:border file:border-border file:bg-surface-2 file:px-3 file:py-2 file:text-text"
            />
          </Field>
        </div>
      </Card>
      {error && <p className="text-danger">{error}</p>}
      <div className="flex gap-3">
        <PrimaryButton disabled={save.isPending}>{save.isPending ? "Saving…" : "Save"}</PrimaryButton>
        <SecondaryButton onClick={() => nav(isNew ? "/vehicles" : `/vehicles/${id}`)}>Cancel</SecondaryButton>
      </div>
    </form>
  );
}
