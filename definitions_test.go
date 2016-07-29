package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"testing"
)

func TestResourceDefinitions(t *testing.T) {
	cmdFactory := func(project, resource string) *exec.Cmd {
		return helperCommand("echo", fmt.Sprintf(`{"project": %q, "resource": %q}`, project, resource))
	}
	tests := []struct {
		project                string
		types                  []string
		wantStdout, wantStderr string
	}{
		{
			project: "test-project",
			types:   []string{"svc", "pod", "dc"},
			wantStdout: `{"project": "test-project", "resource": "svc"}
{"project": "test-project", "resource": "pod"}
{"project": "test-project", "resource": "dc"}
`,
			wantStderr: "",
		},
		// TODO: Add tests involving error cases.
	}
	for _, tt := range tests {
		var stdout, stderr bytes.Buffer
		outFor := func(project, resource string) (io.Writer, io.Closer, error) {
			return &stdout, ioutil.NopCloser(nil), nil
		}
		errOutFor := func(project, resource string) (io.Writer, io.Closer, error) {
			return &stderr, ioutil.NopCloser(nil), nil
		}
		task := resourceDefinitions(cmdFactory, tt.project, tt.types, outFor, errOutFor)

		if err := task(); err != nil {
			t.Errorf("task failed: %v", err)
		}
		if got := stdout.String(); got != tt.wantStdout {
			t.Errorf("stdout = %q, want %q", got, tt.wantStdout)
		}
		if got := stderr.String(); got != tt.wantStderr {
			t.Errorf("stderr = %q, want %q", got, tt.wantStderr)
		}
	}
}
