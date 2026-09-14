package brand

import (
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
	return []map[string]any{
		{"src": SVGURL(rootURL), "mimeType": "image/svg+xml"},
		{"src": PNGURL(rootURL), "mimeType": "image/png"},
	}
}

func Link(rootURL string) string {
	return `<` + SVGURL(rootURL) + `>; rel="icon"; type="image/svg+xml", <` + PNGURL(rootURL) + `>; rel="icon"; type="image/png"`
}

func ServeSVG(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(SVG)
}

func ServePNG(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(PNG)
}
