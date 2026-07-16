import { useEffect, useState } from "react";
import { applyUpdate, getPendingUpdate } from "../lib/sw";

/** Shows "Update available" when a new service worker is waiting. */
export default function UpdateToast() {
  const [worker, setWorker] = useState<ServiceWorker | null>(getPendingUpdate());

  useEffect(() => {
    const on = (e: Event) => setWorker((e as CustomEvent<ServiceWorker>).detail);
    window.addEventListener("vmt-sw-update", on);
    return () => window.removeEventListener("vmt-sw-update", on);
  }, []);

  if (!worker) return null;
  return (
    <div className="fixed inset-x-4 bottom-[calc(4.5rem+env(safe-area-inset-bottom))] z-40 mx-auto flex max-w-md items-center gap-3 rounded-xl border border-border bg-surface p-3 shadow-lg md:bottom-4">
      <span className="min-w-0 flex-1 text-sm">A new version of VMT is available.</span>
      <button
        onClick={() => applyUpdate(worker)}
        className="min-h-10 shrink-0 rounded-lg bg-primary px-4 text-sm font-semibold text-white hover:bg-primary-strong"
      >
        Reload
      </button>
    </div>
  );
}
