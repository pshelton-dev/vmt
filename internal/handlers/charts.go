package handlers

import (
	"fmt"
	"html"
	"html/template"
	"strings"
)

// barChart renders a simple responsive vertical bar chart as inline SVG.
// labels and values must be the same length. valueFmt formats the tooltip/label
// for each value (e.g. a currency formatter).
func barChart(labels []string, values []float64, valueFmt func(float64) string) template.HTML {
	if len(values) == 0 {
		return ""
	}
	const (
		w      = 720.0
		h      = 240.0
		padL   = 8.0
		padR   = 8.0
		padTop = 16.0
		padBot = 28.0
	)
	max := 0.0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		max = 1
	}
	plotW := w - padL - padR
	plotH := h - padTop - padBot
	n := float64(len(values))
	slot := plotW / n
	barW := slot * 0.6
	gap := (slot - barW) / 2

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="chart" viewBox="0 0 %.0f %.0f" preserveAspectRatio="xMidYMid meet" role="img">`, w, h)
	// baseline
	fmt.Fprintf(&b, `<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`,
		padL, padTop+plotH, w-padR, padTop+plotH)
	for i, v := range values {
		bh := (v / max) * plotH
		x := padL + float64(i)*slot + gap
		y := padTop + (plotH - bh)
		fmt.Fprintf(&b, `<rect class="bar" x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="2"><title>%s: %s</title></rect>`,
			x, y, barW, bh, html.EscapeString(labels[i]), html.EscapeString(valueFmt(v)))
		// label under bar
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle">%s</text>`,
			x+barW/2, h-10, html.EscapeString(labels[i]))
	}
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}
