import { useQuery } from "@tanstack/react-query";

// The bundled make/model dataset lives in the Go binary's static files
// (shared with the v1 form). Free text is always allowed; this only feeds
// datalist suggestions.
export function useVehicleData() {
  return useQuery({
    queryKey: ["vehicle-data"],
    staleTime: Infinity,
    queryFn: async (): Promise<Record<string, string[]>> => {
      const res = await fetch("/static/vehicle-data.json");
      if (!res.ok) return {};
      return res.json();
    },
  });
}

/** Model-year options: next year down to 1950. */
export function yearOptions(): number[] {
  const next = new Date().getFullYear() + 1;
  const out: number[] = [];
  for (let y = next; y >= 1950; y--) out.push(y);
  return out;
}
