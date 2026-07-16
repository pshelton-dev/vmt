/* VMT service worker.
 *
 * Strategy:
 *  - /api/*           -> network only (live data; the app handles offline errors)
 *  - hashed assets    -> cache-first (immutable: Vite content-hashes filenames)
 *  - navigations etc. -> network-first with cache fallback (index.html is
 *                        no-cache from the server, so updates flow through)
 *
 * The cache name is versioned per deploy via the ?v= query the registration
 * passes; old caches are dropped on activate. A waiting worker takes over
 * when the page posts SKIP_WAITING (the "update available" toast).
 */
const VERSION = new URL(self.location.href).searchParams.get("v") || "dev";
const CACHE = `vmt-${VERSION}`;

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE).then((c) => c.addAll(["/app/", "/app/manifest.webmanifest"])),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("message", (event) => {
  if (event.data === "SKIP_WAITING") self.skipWaiting();
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);
  if (event.request.method !== "GET" || url.origin !== self.location.origin) return;
  if (url.pathname.startsWith("/api/")) return; // network only

  // Immutable hashed assets: cache-first.
  if (url.pathname.startsWith("/app/assets/")) {
    event.respondWith(
      caches.match(event.request).then(
        (hit) =>
          hit ||
          fetch(event.request).then((res) => {
            const copy = res.clone();
            caches.open(CACHE).then((c) => c.put(event.request, copy));
            return res;
          }),
      ),
    );
    return;
  }

  // Everything else under /app and /static: network-first, cache fallback.
  if (url.pathname.startsWith("/app") || url.pathname.startsWith("/static/")) {
    event.respondWith(
      fetch(event.request)
        .then((res) => {
          const copy = res.clone();
          caches.open(CACHE).then((c) => c.put(event.request, copy));
          return res;
        })
        .catch(() => caches.match(event.request).then((hit) => hit || caches.match("/app/"))),
    );
  }
});
