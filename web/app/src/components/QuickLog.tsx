import { useEffect, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";
import { api, ApiError, type Meta, type ServiceRecord, type Vehicle } from "../lib/api";
import { Field, Select, TextInput } from "./form";

/**
 * Quick-log: a floating action button (phones only) that opens a bottom sheet
 * for fast at-the-shop service entry. The vehicle is pre-selected when you're
 * on that vehicle's page; date defaults to today.
 */
export default function QuickLog() {
  const [open, setOpen] = useState(false);
  const location = useLocation();

  // Hide the FAB where it would collide with form actions.
  const onForm = /\/(new|edit)$/.test(location.pathname);
  if (onForm) return null;

  return (
    <>
      {!open && (
        <button
          onClick={() => setOpen(true)}
          aria-label="Quick-log a service"
          className="fixed bottom-[calc(4.5rem+env(safe-area-inset-bottom))] right-4 z-20 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-2xl font-bold text-white shadow-lg hover:bg-primary-strong md:hidden"
        >
          +
        </button>
      )}
      {open && <QuickLogSheet close={() => setOpen(false)} />}
    </>
  );
}

function QuickLogSheet({ close }: { close: () => void }) {
  const qc = useQueryClient();
  const location = useLocation();
  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });
  const vehicles = useQuery({ queryKey: ["vehicles"], queryFn: () => api.get<Vehicle[]>("/vehicles") });

  // Pre-select the vehicle whose page we're on (/vehicles/:id...).
  const routeVehicle = location.pathname.match(/^\/vehicles\/(\d+)/)?.[1] ?? "";

  const [vehicleId, setVehicleId] = useState(routeVehicle);
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [odometer, setOdometer] = useState("");
  const [category, setCategory] = useState("Oil Change");
  const [description, setDescription] = useState("");
  const [cost, setCost] = useState("");
  const [error, setError] = useState("");

  // Default to the first vehicle once loaded; prefill odometer.
  useEffect(() => {
    if (!vehicleId && vehicles.data?.length) setVehicleId(String(vehicles.data[0].id));
  }, [vehicleId, vehicles.data]);
  useEffect(() => {
    const v = vehicles.data?.find((x) => String(x.id) === vehicleId);
    if (v && v.odometer > 0) setOdometer(String(v.odometer));
  }, [vehicleId, vehicles.data]);

  const save = useMutation({
    mutationFn: () =>
      api.post<ServiceRecord>(`/vehicles/${vehicleId}/services`, {
        date,
        odometer: odometer ? Number(odometer) : null,
        category,
        description,
        vendor: "",
        cost: cost ? Number(cost) : 0,
        notes: "",
      }),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Save failed."),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError("");
    save.mutate();
  };

  return (
    <div className="fixed inset-0 z-30 md:hidden" role="dialog" aria-label="Quick-log service">
      <div className="absolute inset-0 bg-black/50" onClick={close} />
      <form
        onSubmit={submit}
        className="absolute inset-x-0 bottom-0 space-y-4 rounded-t-2xl border-t border-border bg-surface p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]"
      >
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold">Log service</h2>
          <button type="button" onClick={close} aria-label="Close" className="flex h-11 w-11 items-center justify-center text-xl text-muted">
            ✕
          </button>
        </div>
        <Field label="Vehicle">
          <Select value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
            {(vehicles.data ?? []).map((v) => (
              <option key={v.id} value={v.id}>{v.name}</option>
            ))}
          </Select>
        </Field>
        <Field label="Description *">
          <TextInput value={description} onChange={(e) => setDescription(e.target.value)} required placeholder="Oil change" />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Date">
            <TextInput type="date" value={date} onChange={(e) => setDate(e.target.value)} required />
          </Field>
          <Field label="Category">
            <Select value={category} onChange={(e) => setCategory(e.target.value)}>
              {(meta.data?.service_categories ?? ["Other"]).map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </Select>
          </Field>
          <Field label="Odometer">
            <TextInput type="number" min={0} value={odometer} onChange={(e) => setOdometer(e.target.value)} inputMode="numeric" />
          </Field>
          <Field label="Cost">
            <TextInput type="number" min={0} step="0.01" value={cost} onChange={(e) => setCost(e.target.value)} inputMode="decimal" />
          </Field>
        </div>
        {error && <p className="text-sm text-danger">{error}</p>}
        <button
          type="submit"
          disabled={save.isPending || !vehicleId}
          className="min-h-12 w-full rounded-lg bg-primary font-semibold text-white hover:bg-primary-strong disabled:opacity-50"
        >
          {save.isPending ? "Saving…" : "Save"}
        </button>
      </form>
    </div>
  );
}
