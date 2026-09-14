package catalog

import (
	"testing"

	"acahti/internal/config"
)

func TestAcahtiCheckURL(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com"}, nil, nil)
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://acahti.example.com/ci", "https://acahti.example.com/pipelines"},
		{"https://acahti.example.com/ci/", "https://acahti.example.com/pipelines"},
		{"https://acahti.example.com/ci/acme/demo/12", "https://acahti.example.com/pipelines/acme/demo/12"},
		{"https://acahti.example.com/ci/repos/acme/demo/pipeline/12", "https://acahti.example.com/pipelines/acme/demo/12"},
		{"https://acahti.example.com/repos/acme/demo", "https://acahti.example.com/repos/acme/demo"},
	}
	for _, tc := range cases {
		if got := c.acahtiCheckURL(tc.in); got != tc.want {
			t.Fatalf("acahtiCheckURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
