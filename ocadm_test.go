package main

import (
	"reflect"
	"testing"
)

func TestGetOcAdmDiagnosticsTask(t *testing.T) {
	expectedCalls := []RunCall{
		{
			[]string{"oc", "adm", "diagnostics"},
			"oc_adm_diagnostics",
		},
	}

	runner := &FakeRunner{}

	task := GetOcAdmDiagnosticsTask(runner)

	if err := task(); err != nil {
		t.Errorf("task() = %v, want %v", err, nil)
	}
	if !reflect.DeepEqual(runner.Calls, expectedCalls) {
		t.Errorf("runner.Calls = %q, want %q", runner.Calls, expectedCalls)
	}
}
