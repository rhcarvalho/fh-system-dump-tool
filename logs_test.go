package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestFetchLogs(t *testing.T) {
	echoCmdFactory := func(resource LoggableResource) *exec.Cmd {
		return helperCommand("echo", fmt.Sprintf(`%+v`, resource))
	}
	falseCmdFactory := func(resource LoggableResource) *exec.Cmd {
		return helperCommand("stderrfail")
	}
	tests := []struct {
		cmdFactory             logsCmdFactory
		resource               LoggableResource
		shouldFail             bool
		wantStdout, wantStderr string
	}{
		{
			cmdFactory: echoCmdFactory,
			resource: LoggableResource{
				Project:   "test-project",
				Type:      "pod",
				Name:      "pod-1",
				Container: "container-1",
			},
			wantStdout: "{Project:test-project Type:pod Name:pod-1 Container:container-1}\n",
			wantStderr: "",
		},
		{
			cmdFactory: echoCmdFactory,
			resource: LoggableResource{
				Project: "test-project",
				Type:    "dc",
				Name:    "dc-1",
			},
			wantStdout: "{Project:test-project Type:dc Name:dc-1 Container:}\n",
			wantStderr: "",
		},
		{
			cmdFactory: falseCmdFactory,
			resource: LoggableResource{
				Project: "test-project",
				Type:    "dc",
				Name:    "dc-1",
			},
			shouldFail: true,
			wantStdout: "",
			wantStderr: "some stderr text\n",
		},
	}
	for _, tt := range tests {
		var stdout, stderr bytes.Buffer
		task := fetchLogs(tt.cmdFactory, tt.resource, &stdout, &stderr)

		err := task()
		if (err != nil) != tt.shouldFail {
			want := "nil"
			if tt.shouldFail {
				want = "not nil"
			}
			t.Errorf("task() = %v, want %v", err, want)
		}
		if tt.shouldFail && !strings.Contains(err.Error(), tt.wantStderr) {
			t.Errorf("error message doesn't include stderr:\ngot: %v\nwant: %v", err, tt.wantStderr)
		}
		if got := stdout.String(); got != tt.wantStdout {
			t.Errorf("stdout = %q, want %q", got, tt.wantStdout)
		}
		if got := stderr.String(); got != tt.wantStderr {
			t.Errorf("stderr = %q, want %q", got, tt.wantStderr)
		}
	}
}
