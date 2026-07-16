import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { api, type ReferenceItem } from "../../lib/api";
import { Card, Field, Select, TextInput } from "../../components/form";

interface RefInput {
  kind: string;
  name: string;
  part_number: string;
  manufacturer: string;
  capacity: string;
  spec: string;
  notes: string;
}

const blank: RefInput = {
  kind: "part", name: "", part_number: "", manufacturer: "", capacity: "", spec: "", notes: "",
};

/** Reference & specs tab: parts and fluids tables plus inline add/edit. */
export default function ReferenceEditor({
  vehicleId, items, refresh,
}: { vehicleId: number; items: ReferenceItem[]; refresh: () => void }) {
  const [form, setForm] = useState<RefInput>(blank);
  const [editing, setEditing] = useState<number | null>(null);

  const set = (k: keyof RefInput) => (e: { target: { value: string } }) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const save = useMutation({
    mutationFn: () =>
      editing
        ? api.put<ReferenceItem>(`/reference/${editing}`, form)
        : api.post<ReferenceItem>(`/vehicles/${vehicleId}/reference`, form),
    onSuccess: () => {
      setForm(blank);
      setEditing(null);
      refresh();
    },
  });
  const del = useMutation({
    mutationFn: (rid: number) => api.delete(`/reference/${rid}`),
    onSuccess: refresh,
  });

  const startEdit = (ri: ReferenceItem) => {
    setEditing(ri.id);
    setForm({
      kind: ri.kind, name: ri.name, part_number: ri.part_number,
      manufacturer: ri.manufacturer, capacity: ri.capacity, spec: ri.spec, notes: ri.notes,
    });
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (form.name.trim()) save.mutate();
  };

  const parts = items.filter((i) => i.kind !== "fluid");
  const fluids = items.filter((i) => i.kind === "fluid");

  const row = (ri: ReferenceItem, cols: string[]) => (
    <li key={ri.id} className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5 py-2.5">
      <div className="min-w-0 flex-1">
        <span className="font-medium">{ri.name}</span>
        <div className="text-sm text-muted">
          {cols.filter(Boolean).join(" · ") || "—"}
        </div>
        {ri.notes && <div className="text-sm text-muted">{ri.notes}</div>}
      </div>
      <div className="flex shrink-0 gap-1">
        <button onClick={() => startEdit(ri)} className="flex min-h-9 items-center px-2 text-sm text-muted hover:text-text">
          edit
        </button>
        <button
          onClick={() => confirm("Delete this item?") && del.mutate(ri.id)}
          className="flex min-h-9 min-w-9 items-center justify-center text-sm text-muted hover:text-danger"
        >
          ✕
        </button>
      </div>
    </li>
  );

  return (
    <div className="space-y-4">
      <Card title="Parts & filters">
        {parts.length === 0 ? (
          <p className="py-4 text-center text-muted">No parts listed yet.</p>
        ) : (
          <ul className="divide-y divide-border">
            {parts.map((ri) =>
              row(ri, [
                ri.part_number && `# ${ri.part_number}`,
                ri.manufacturer,
                ri.spec,
              ] as string[]),
            )}
          </ul>
        )}
      </Card>

      <Card title="Fluids & capacities">
        {fluids.length === 0 ? (
          <p className="py-4 text-center text-muted">No fluids listed yet.</p>
        ) : (
          <ul className="divide-y divide-border">
            {fluids.map((ri) =>
              row(ri, [
                ri.capacity && `capacity ${ri.capacity}`,
                ri.spec,
                ri.manufacturer,
                ri.part_number && `# ${ri.part_number}`,
              ] as string[]),
            )}
          </ul>
        )}
      </Card>

      <Card title={editing ? "Edit reference item" : "Add reference item"}>
        <form onSubmit={submit} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Type">
              <Select value={form.kind} onChange={set("kind")}>
                <option value="part">Part / filter</option>
                <option value="fluid">Fluid</option>
              </Select>
            </Field>
            <Field label="Name *">
              <TextInput value={form.name} onChange={set("name")} required placeholder="Oil filter / Engine oil" />
            </Field>
            <Field label="Part number">
              <TextInput value={form.part_number} onChange={set("part_number")} placeholder="FL-2016" />
            </Field>
            <Field label="Manufacturer / brand">
              <TextInput value={form.manufacturer} onChange={set("manufacturer")} placeholder="Motorcraft" />
            </Field>
            <Field label="Capacity (fluids)">
              <TextInput value={form.capacity} onChange={set("capacity")} placeholder="13 qt" />
            </Field>
            <Field label="Spec / grade">
              <TextInput value={form.spec} onChange={set("spec")} placeholder="15W-40" />
            </Field>
          </div>
          <Field label="Notes">
            <TextInput value={form.notes} onChange={set("notes")} placeholder="optional" />
          </Field>
          <div className="flex gap-3">
            <button type="submit" className="min-h-11 rounded-lg bg-primary px-5 font-semibold text-white hover:bg-primary-strong">
              {editing ? "Save changes" : "+ Add item"}
            </button>
            {editing && (
              <button
                type="button"
                onClick={() => { setEditing(null); setForm(blank); }}
                className="min-h-11 rounded-lg border border-border px-5 hover:border-muted"
              >
                Cancel
              </button>
            )}
          </div>
        </form>
      </Card>
    </div>
  );
}
