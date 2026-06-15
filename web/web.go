// Package web embeds the templates and static assets into the binary.
package web

import "embed"

//go:embed templates/*.html
var TemplatesFS embed.FS

//go:embed static/*
var StaticFS embed.FS
