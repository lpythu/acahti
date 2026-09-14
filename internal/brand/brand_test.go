package brand

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIcons(t *testing.T) {
	icons := Icons("https://acahti.example/")
	if len(icons) != 2 {
		t.Fatalf("len=%d", len(icons))
	}
	src, _ := icons[0]["src"].(string)
	if !strings.HasPrefix(src, "data:image/png;base64,") || icons[0]["mimeType"] != "image/png" {
		t.Fatalf("data=%v", icons[0])
	}
	if icons[1]["src"] != "https://acahti.example/acahti.png" || icons[1]["mimeType"] != "image/png" {
		t.Fatalf("png=%v", icons[1])
	}
	if _, ok := icons[0]["sizes"]; ok {
		t.Fatal("omit sizes; older clients reject array vs string")
	}
}

func TestServeBrand(t *testing.T) {
	rr := httptest.NewRecorder()
	ServeSVG(rr, httptest.NewRequest(http.MethodGet, "/acahti.svg", nil))
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("svg %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}
	if !strings.Contains(rr.Body.String(), "<svg") {
		t.Fatal("svg body")
	}
	rr = httptest.NewRecorder()
	ServePNG(rr, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
	if rr.Code != http.StatusOK || rr.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("png %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}
	if len(rr.Body.Bytes()) < 8 || string(rr.Body.Bytes()[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("png magic")
	}
}
