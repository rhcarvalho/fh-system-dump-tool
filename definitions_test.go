package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestResourceDefinition(t *testing.T) {
	tests := []struct {
		project  string
		resource string
		calls    []RunCall
	}{
		{
			project:  "test-project",
			resource: "deploymentconfigs",
			calls: []RunCall{
				{
					[]string{"oc", "-n", "test-project", "get", "deploymentconfigs", "-o=json"},
					filepath.Join("projects", "test-project", "definitions", "deploymentconfigs.json"),
				},
			},
		},
		{
			project:  "test-project",
			resource: "svc/mongodb-1",
			calls: []RunCall{
				{
					[]string{"oc", "-n", "test-project", "get", "svc/mongodb-1", "-o=json"},
					filepath.Join("projects", "test-project", "definitions", "svc_mongodb-1.json"),
				},
			},
		},
	}
	for i, tt := range tests {
		runner := &FakeRunner{}
		task := ResourceDefinition(runner, tt.project, tt.resource)
		if err := task(); err != nil {
			t.Errorf("test %d: task() = %v, want %v", i, err, nil)
		}
		if !reflect.DeepEqual(runner.Calls, tt.calls) {
			t.Errorf("test %d: runner.Calls = %q, want %q", i, runner.Calls, tt.calls)
		}
	}
}
