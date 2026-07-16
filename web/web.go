// Package web embeds the static assets and the built SPA into the binary.
package web

import "embed"

//go:embed static/*
var StaticFS embed.FS

// AppFS holds the built SPA (web/app/dist). When the SPA hasn't been built,
// only the committed placeholder.html is present (kept in web/app/public so
// every Vite build re-emits it, keeping git clean).
//
//go:embed all:app/dist
var AppFS embed.FS
