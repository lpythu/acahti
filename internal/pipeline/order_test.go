package pipeline

import (
	"reflect"
	"testing"
)

func TestJobName(t *testing.T) {
	if got := JobName(".acahti/pipelines/cd.office.yaml"); got != "cd.office" {
		t.Fatalf("%q", got)
	}
}

func TestDependsOn(t *testing.T) {
	if got := DependsOn("depends_on: [ci]\n"); !reflect.DeepEqual(got, []string{"ci"}) {
		t.Fatalf("%v", got)
	}
	if got := DependsOn("depends_on: cd.office\n"); !reflect.DeepEqual(got, []string{"cd.office"}) {
		t.Fatalf("%v", got)
	}
	if got := DependsOn("steps: {}\n"); got != nil {
		t.Fatalf("%v", got)
	}
}
