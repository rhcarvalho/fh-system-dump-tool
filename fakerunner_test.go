package main

import (
	"os/exec"
	"sync"
)

type FakeRunner struct {
	mu    sync.Mutex
	Calls []RunCall
}

type RunCall struct {
	args []string
	path string
}

func (r *FakeRunner) Run(cmd *exec.Cmd, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls = append(r.Calls, RunCall{cmd.Args, path})
	return nil
}

var _ Runner = (*FakeRunner)(nil)
