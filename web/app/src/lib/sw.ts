// Service-worker registration + update detection. When a new worker is
// installed while an old one controls the page, we dispatch an event the
// Layout turns into an "Update available" toast; accepting posts SKIP_WAITING
// and reloads once the new worker takes control.

declare const __BUILD_ID__: string;

// The waiting worker is stashed here so a toast mounting *after* the event
// fired (e.g. the user was still on the login screen) can still pick it up.
let pending: ServiceWorker | null = null;

export function getPendingUpdate() {
  return pending;
}

export function registerSW() {
  if (!("serviceWorker" in navigator) || !import.meta.env.PROD) return;

  window.addEventListener("load", async () => {
    try {
      const reg = await navigator.serviceWorker.register(`/sw.js?v=${__BUILD_ID__}`, {
        scope: "/",
      });

      const notify = (worker: ServiceWorker) => {
        pending = worker;
        window.dispatchEvent(new CustomEvent("vmt-sw-update", { detail: worker }));
      };
      // A worker already waiting (page reopened after a deploy).
      if (reg.waiting && navigator.serviceWorker.controller) notify(reg.waiting);
      reg.addEventListener("updatefound", () => {
        const w = reg.installing;
        w?.addEventListener("statechange", () => {
          if (w.state === "installed" && navigator.serviceWorker.controller) notify(w);
        });
      });

      let reloaded = false;
      navigator.serviceWorker.addEventListener("controllerchange", () => {
        if (!reloaded) {
          reloaded = true;
          window.location.reload();
        }
      });
    } catch {
      // SW is progressive enhancement; the app works fine without it.
    }
  });
}

export function applyUpdate(worker: ServiceWorker) {
  worker.postMessage("SKIP_WAITING");
}
