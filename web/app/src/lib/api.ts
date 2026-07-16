// Thin fetch wrapper for the VMT JSON API. Session auth rides on the
// vmt_session cookie (same origin), so there is no token handling here.

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, init);
  if (res.status === 204) return undefined as T;
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(res.status, body?.error ?? `HTTP ${res.status}`);
  }
  return body as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  put: <T>(path: string, body: unknown) =>
    request<T>(path, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),

  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),

  /** multipart upload; `field` is the server-expected file field name */
  upload: <T>(path: string, field: string, file: File) => {
    const form = new FormData();
    form.append(field, file);
    return request<T>(path, { method: "POST", body: form });
  },
};

// ---- API types (mirror the Go models' JSON) ----

export interface Vehicle {
  id: number;
  name: string;
  make: string;
  model: string;
  year: number | null;
  vin: string;
  license_plate: string;
  color: string;
  odometer: number;
  purchase_date: string | null;
  notes: string;
  photo_id: number | null;
  total_cost: number;
  service_count: number;
  due_reminders: number;
}

export interface Attachment {
  id: number;
  vehicle_id: number | null;
  service_id: number | null;
  kind: "photo" | "document";
  original_name: string;
  content_type: string;
  size: number;
}

export interface ServiceRecord {
  id: number;
  vehicle_id: number;
  date: string;
  odometer: number | null;
  category: string;
  description: string;
  vendor: string;
  cost: number;
  notes: string;
  vehicle_name?: string;
  attachments?: Attachment[];
}

export type ReminderStatus = "ok" | "soon" | "due" | "overdue";

export interface Reminder {
  id: number;
  vehicle_id: number;
  title: string;
  due_date: string | null;
  due_odometer: number | null;
  interval_months: number | null;
  interval_miles: number | null;
  notes: string;
  completed: boolean;
  notify: boolean;
  last_notified: string | null;
  vehicle_name?: string;
  status?: ReminderStatus;
  status_text?: string;
}

export interface ReferenceItem {
  id: number;
  vehicle_id: number;
  kind: "part" | "fluid";
  name: string;
  part_number: string;
  manufacturer: string;
  capacity: string;
  spec: string;
  notes: string;
  position: number;
}

export interface Dashboard {
  vehicles: Vehicle[];
  due_reminders: Reminder[];
  recent_services: ServiceRecord[];
  total_cost: number;
  service_count: number;
}

export interface SessionInfo {
  configured: boolean;
  authed: boolean;
}

export interface Meta {
  service_categories: string[];
  reference_kinds: { value: string; label: string }[];
  date_presets: { layout: string; example: string }[];
  currency: string;
  distance_unit: string;
  date_format: string;
}

export interface Settings {
  currency: string;
  distance_unit: string;
  date_format: string;
  smtp_configured: boolean;
  smtp_from: string;
  notify_enabled: boolean;
  notify_email: string;
}
