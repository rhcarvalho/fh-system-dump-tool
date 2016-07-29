package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

// helperCommand creates a simulated external command for tests.
func helperCommand(name string, arg ...string) *exec.Cmd {
	arg = append([]string{"-test.run=TestHelperProcess", "--", name}, arg...)
	cmd := exec.Command(os.Args[0], arg...)
	cmd.Env = []string{"WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess isn't a real test. It's used by other tests to simulate
// running external commands.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		iargs := []interface{}{}
		for _, s := range args {
			iargs = append(iargs, s)
		}
		fmt.Println(iargs...)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func TestGetSpaceSeparated(t *testing.T) {
	tests := []struct {
		projects []string
		want     []string
	}{
		{},
		{[]string{"foo", "bar"}, []string{"foo", "bar"}},
		{[]string{"foo", "bar", ""}, []string{"foo", "bar"}},
		// TODO: Add tests involving error cases.
	}
	for _, tt := range tests {
		cmd := helperCommand("echo", tt.projects...)
		got, err := getSpaceSeparated(cmd)
		if err != nil {
			t.Errorf("getSpaceSeparated(%v) returned non-nil error: %v", cmd.Args, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("getSpaceSeparated(%v) = %v, want %v", cmd.Args, got, tt.want)
		}
	}
}
