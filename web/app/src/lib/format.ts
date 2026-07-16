// Display formatting helpers. Currency/unit come from /api/v1/meta.

export function money(currency: string, v: number): string {
  return (
    currency +
    v.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  );
}

export function miles(unit: string, v: number): string {
  return `${v.toLocaleString("en-US")} ${unit}`;
}

/** ISO yyyy-mm-dd -> "Jul 15, 2026" (SPA uses one friendly format). */
export function formatDate(iso: string): string {
  const [y, m, d] = iso.split("-").map(Number);
  if (!y || !m || !d) return iso;
  return new Date(y, m - 1, d).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
