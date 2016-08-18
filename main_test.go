package main

import (
	"fmt"
	"os"
	"os/exec"
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
	case "stderrfail":
		fmt.Fprintf(os.Stderr, "some stderr text\n")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}
