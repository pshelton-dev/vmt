package handlers

// referenceKinds are the selectable item types (exposed via /api/v1/meta).
var referenceKinds = []struct{ Value, Label string }{
	{"part", "Part / filter"},
	{"fluid", "Fluid"},
}
