package main

import (
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type RunFunc func(cmd *exec.Cmd, path string) error

type LogsFakeRunner struct {
	// callMap maps commands to mock functions.
	callMap map[string]RunFunc

	mu sync.Mutex
	// Seen is a set of commands that were run.
	Seen map[string]struct{}
	// Unhandled stores commands that were run and not found in callMap.
	Unhandled []string
}

func NewLogsFakeRunner(callMap map[string]RunFunc) *LogsFakeRunner {
	return &LogsFakeRunner{
		callMap: callMap,
		Seen:    make(map[string]struct{}),
	}
}

func (r *LogsFakeRunner) Run(cmd *exec.Cmd, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	args := strings.Join(cmd.Args, " ")
	r.Seen[args] = struct{}{}
	var f RunFunc
	for k, v := range r.callMap {
		if strings.Contains(args, k) {
			f = v
		}
	}
	if f != nil {
		return f(cmd, path)
	}
	r.Unhandled = append(r.Unhandled, args)
	return nil
}

func TestGetFetchLogsTasks(t *testing.T) {
	tasks := make(chan Task)
	runner := NewLogsFakeRunner(map[string]RunFunc{
		"get pods": func(cmd *exec.Cmd, path string) error {
			fmt.Fprintln(cmd.Stdout, "pod-1", "pod-2")
			return nil
		},
		"get pod pod-1": func(cmd *exec.Cmd, path string) error {
			fmt.Fprintln(cmd.Stdout, "c11", "c12")
			return nil
		},
		"get pod pod-2": func(cmd *exec.Cmd, path string) error {
			fmt.Fprintln(cmd.Stdout, "c21")
			return nil
		},
		"logs pods/pod-": func(cmd *exec.Cmd, path string) error {
			return nil
		},
	})
	projects := []string{"test-project"}
	resources := []string{"pods"}
	const maxLines = 42
	go func() {
		defer close(tasks)
		GetFetchLogsTasks(tasks, runner, projects, resources, maxLines)
	}()

	i := 0
	for task := range tasks {
		i++
		if err := task(); err != nil {
			t.Errorf("task %d: task() = %v, want %v", i, err, nil)
		}
	}

	calls := map[string]struct{}{
		"oc -n test-project get pods -o=jsonpath={.items[*].metadata.name}":       {},
		"oc -n test-project get pod pod-1 -o=jsonpath={.spec.containers[*].name}": {},
		"oc -n test-project get pod pod-2 -o=jsonpath={.spec.containers[*].name}": {},
		"oc -n test-project logs pods/pod-1 -c c11 --tail 42":                     {},
		"oc -n test-project logs pods/pod-1 -c c11 --tail 42 --previous":          {},
		"oc -n test-project logs pods/pod-1 -c c12 --tail 42":                     {},
		"oc -n test-project logs pods/pod-1 -c c12 --tail 42 --previous":          {},
		"oc -n test-project logs pods/pod-2 -c c21 --tail 42":                     {},
		"oc -n test-project logs pods/pod-2 -c c21 --tail 42 --previous":          {},
	}
	if !reflect.DeepEqual(runner.Seen, calls) {
		t.Errorf("runner.Calls = %q, want %q", runner.Seen, calls)
	}
	if runner.Unhandled != nil {
		t.Logf("unhandled commands:\n%s", strings.Join(runner.Unhandled, "\n"))
	}
}
