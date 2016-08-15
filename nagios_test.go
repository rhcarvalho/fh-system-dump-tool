package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetNagiosStatusData(t *testing.T) {
	tests := []struct {
		project string
		pod     string
		calls   []RunCall
	}{
		{
			project: "test-project",
			pod:     "pod",
			calls: []RunCall{
				{
					[]string{"oc", "exec", "pod", "--", "cat", "/var/log/nagios/status.dat"},
					filepath.Join("projects", "test-project", "nagios", "pod_status.dat"),
				},
			},
		},
	}
	for i, tt := range tests {
		runner := &FakeRunner{}
		task := GetNagiosStatusData(runner, tt.project, tt.pod)
		if err := task(); err != nil {
			t.Errorf("test %d: task() = %v, want %v", i, err, nil)
		}
		if !reflect.DeepEqual(runner.Calls, tt.calls) {
			t.Errorf("test %d: runner.Calls = %q, want %q", i, runner.Calls, tt.calls)
		}
	}
}
func TestGetNagiosHistoricalData(t *testing.T) {
	tests := []struct {
		project string
		pod     string
		calls   []RunCall
	}{
		{
			project: "test-project",
			pod:     "pod",
			calls: []RunCall{
				{
					[]string{"oc", "exec", "pod", "--", "tar", "-c", "-C", "/var/log/nagios", "archives"},
					filepath.Join("projects", "test-project", "nagios", "pod_history.tar"),
				},
			},
		},
	}
	for i, tt := range tests {
		runner := &FakeRunner{}
		task := GetNagiosHistoricalData(runner, tt.project, tt.pod)
		if err := task(); err != nil {
			t.Errorf("test %d: task() = %v, want %v", i, err, nil)
		}
		if !reflect.DeepEqual(runner.Calls, tt.calls) {
			t.Errorf("test %d: runner.Calls = %q, want %q", i, runner.Calls, tt.calls)
		}
	}
}
