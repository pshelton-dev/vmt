import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { api, type Meta, type ServiceRecord, type Vehicle } from "../../lib/api";
import { formatDate, miles, money } from "../../lib/format";

/**
 * A print-ready maintenance record to hand to a buyer. Rendered outside the app
 * shell so there is no navigation to hide; the browser's print dialog produces
 * paper or a PDF. Owner-only details (plate, purchase date, notes, reminders)
 * are deliberately left out.
 */
export default function Report() {
  const { id } = useParams();
  const [showCosts, setShowCosts] = useState(false);

  const meta = useQuery({ queryKey: ["meta"], queryFn: () => api.get<Meta>("/meta") });
  const q = useQuery({
    queryKey: ["vehicle", id],
    queryFn: () => api.get<{ vehicle: Vehicle; services: ServiceRecord[] }>(`/vehicles/${id}`),
  });

  const v = q.data?.vehicle;
  const title = v ? [v.year, v.make, v.model].filter(Boolean).join(" ") || v.name : "";

  // The browser offers the document title as the PDF file name.
  useEffect(() => {
    if (!title) return;
    const prev = document.title;
    document.title = `${title} — Maintenance Record`;
    return () => {
      document.title = prev;
    };
  }, [title]);

  if (q.isLoading) return <p className="p-6 text-muted">Loading…</p>;
  if (q.isError || !q.data || !v) return <p className="p-6 text-danger">Vehicle not found.</p>;

  const currency = meta.data?.currency ?? "$";
  const unit = meta.data?.distance_unit ?? "mi";
  const services = [...q.data.services].reverse(); // API is newest-first; print oldest-first
  const total = services.reduce((sum, s) => sum + s.cost, 0);
  const prepared = new Date().toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });

  return (
    <div className="mx-auto max-w-4xl p-6 print:max-w-none print:p-0">
      <div className="mb-6 flex flex-wrap items-center gap-4 print:hidden">
        <Link to={`/vehicles/${v.id}`} className="text-sm text-muted hover:text-text">
          ← Back to vehicle
        </Link>
        <label className="ml-auto flex min-h-11 items-center gap-2.5 text-sm">
          <input
            type="checkbox"
            checked={showCosts}
            onChange={(e) => setShowCosts(e.target.checked)}
            className="h-5 w-5 accent-[var(--primary)]"
          />
          Include costs
        </label>
        <button
          onClick={() => window.print()}
          className="min-h-11 rounded-lg bg-primary px-5 font-semibold text-white hover:bg-primary-strong"
        >
          Print
        </button>
      </div>

      <h1 className="text-2xl font-bold">{title} — Maintenance Record</h1>
      <p className="mt-1 text-sm text-muted">
        {[v.vin && `VIN ${v.vin}`, v.color, `Odometer ${miles(unit, v.odometer)}`, `Prepared ${prepared}`]
          .filter(Boolean)
          .join(" · ")}
      </p>

      {services.length === 0 ? (
        <p className="mt-6 text-muted">No service records.</p>
      ) : (
        <>
          <p className="mt-4 text-sm">
            {services.length} service {services.length === 1 ? "record" : "records"} ·{" "}
            {formatDate(services[0].date)} – {formatDate(services[services.length - 1].date)}
            {showCosts && ` · Total ${money(currency, total)}`}
          </p>

          <table className="mt-4 w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b-2 border-text">
                <th className="py-2 pr-3">Date</th>
                <th className="py-2 pr-3">Odometer</th>
                <th className="py-2 pr-3">Category</th>
                <th className="py-2 pr-3">Work performed</th>
                <th className="py-2 pr-3">Performed by</th>
                {showCosts && <th className="py-2 text-right">Cost</th>}
              </tr>
            </thead>
            <tbody>
              {services.map((s) => (
                <tr key={s.id} className="break-inside-avoid border-b border-border align-top">
                  <td className="whitespace-nowrap py-2 pr-3">{formatDate(s.date)}</td>
                  <td className="whitespace-nowrap py-2 pr-3">
                    {s.odometer != null ? s.odometer.toLocaleString("en-US") : "—"}
                  </td>
                  <td className="whitespace-nowrap py-2 pr-3">{s.category}</td>
                  <td className="py-2 pr-3">{s.description}</td>
                  <td className="py-2 pr-3">{s.vendor || "—"}</td>
                  {showCosts && <td className="whitespace-nowrap py-2 text-right">{money(currency, s.cost)}</td>}
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}
