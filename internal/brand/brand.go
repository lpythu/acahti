package brand

import (
	"encoding/base64"
	_ "embed"
	"net/http"
	"strings"
)

//go:embed acahti.svg
var SVG []byte

//go:embed acahti.png
var PNG []byte

func Root(rootURL string) string {
	return strings.TrimRight(rootURL, "/")
}

func SVGURL(rootURL string) string {
	return Root(rootURL) + "/acahti.svg"
}

func PNGURL(rootURL string) string {
	return Root(rootURL) + "/acahti.png"
}

func Icons(rootURL string) []map[string]any {
	// Cursor cannot decode this 333-byte SVG; send the PNG mark only.
	return []map[string]any{
		{"src": "data:image/png;base64," + base64.StdEncoding.EncodeToString(PNG), "mimeType": "image/png"},
		{"src": PNGURL(rootURL), "mimeType": "image/png"},
	}
}

func Link(rootURL string) string {
	return `<` + SVGURL(rootURL) + `>; rel="icon"; type="image/svg+xml", <` + PNGURL(rootURL) + `>; rel="icon"; type="image/png"`
}

func ServeSVG(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(SVG)
}

func ServePNG(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(PNG)
}
