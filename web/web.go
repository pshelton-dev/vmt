// Package web embeds the templates and static assets into the binary.
package web

import "embed"

//go:embed templates/*.html
var TemplatesFS embed.FS

//go:embed static/*
var StaticFS embed.FS

// AppFS holds the built v2 SPA (web/app/dist). When the SPA hasn't been built,
// only the committed placeholder.html is present (kept in web/app/public so
// every Vite build re-emits it, keeping git clean).
//
//go:embed all:app/dist
var AppFS embed.FS
