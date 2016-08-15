package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestFetchLogs(t *testing.T) {
	tests := []struct {
		resource LoggableResource
		maxLines int
		calls    []RunCall
	}{
		{
			resource: LoggableResource{
				Project:   "test-project",
				Type:      "pod",
				Name:      "pod-1",
				Container: "container-1",
			},
			maxLines: 42,
			calls: []RunCall{
				{
					[]string{"oc", "-n", "test-project", "logs", "pod/pod-1", "-c", "container-1", "--tail", "42"},
					filepath.Join("projects", "test-project", "logs", "pod_pod-1_container-1.logs"),
				},
			},
		},
		{
			resource: LoggableResource{
				Project:   "another-project",
				Type:      "",
				Name:      "supercore",
				Container: "web",
			},
			maxLines: 100,
			calls: []RunCall{
				{
					[]string{"oc", "-n", "another-project", "logs", "supercore", "-c", "web", "--tail", "100"},
					filepath.Join("projects", "another-project", "logs", "supercore_web.logs"),
				},
			},
		},
	}
	for i, tt := range tests {
		runner := &FakeRunner{}
		task := FetchLogs(runner, tt.resource, tt.maxLines)
		if err := task(); err != nil {
			t.Errorf("test %d: task() = %v, want %v", i, err, nil)
		}
		if !reflect.DeepEqual(runner.Calls, tt.calls) {
			t.Errorf("test %d: runner.Calls = %q, want %q", i, runner.Calls, tt.calls)
		}
	}
}
