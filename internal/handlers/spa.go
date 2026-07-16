package handlers

import (
	"io/fs"
	"net/http"
	"strings"

	"vmt/web"
)

// spaHandler serves the built v2 SPA under /app. Real asset paths are served
// as files; anything else falls back to index.html so client-side routes
// (/app/vehicles/3, …) deep-link correctly. If the SPA wasn't built into this
// binary, the committed placeholder page explains how to build it.
func spaHandler() http.Handler {
	dist, err := fs.Sub(web.AppFS, "app/dist")
	if err != nil {
		return http.NotFoundHandler() // embed guarantees the dir; belt & suspenders
	}
	files := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/app")
		path = strings.TrimPrefix(path, "/")
		if path != "" {
			if f, err := dist.Open(path); err == nil {
				f.Close()
				r.URL.Path = "/" + path
				files.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback: serve index.html (or the not-built placeholder).
		name := "index.html"
		if _, err := fs.Stat(dist, name); err != nil {
			name = "placeholder.html"
		}
		body, err := fs.ReadFile(dist, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// index.html must not be cached: it names the hashed asset files, and a
		// stale copy would reference assets that no longer exist after a deploy.
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	})
}
