package order

import (
	"reflect"
	"testing"
)

func TestNamesParentsBeforeChildren(t *testing.T) {
	got := Names(
		[]string{"e2e.office", "cd.office", "ci"},
		map[string][]string{
			"cd.office":  {"ci"},
			"e2e.office": {"cd.office"},
		},
	)
	want := []string{"ci", "cd.office", "e2e.office"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestNamesSameRankByName(t *testing.T) {
	got := Names(
		[]string{"e2e.hk", "e2e.office", "cd.hk", "cd.office", "ci"},
		map[string][]string{
			"cd.office":  {"ci"},
			"cd.hk":      {"ci"},
			"e2e.office": {"cd.office"},
			"e2e.hk":     {"cd.hk"},
		},
	)
	want := []string{"ci", "cd.hk", "cd.office", "e2e.hk", "e2e.office"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
