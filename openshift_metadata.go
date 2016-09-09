package main

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// GetOpenShiftMetadataTasks sends tasks to fetch OpenShift metadata about
// versions, current user and its permissions.
func GetOpenShiftMetadataTasks(tasks chan<- Task, runner Runner, projects []string) {
	argsList := [][]string{
		{"version"},
		{"whoami"},
		{"policy", "can-i", "list", "projects"},
		{"policy", "can-i", "list", "persistentvolumes"},
		{"policy", "can-i", "list", "nodes"},
	}
	for _, args := range argsList {
		args := args
		tasks <- func() error {
			cmd := exec.Command("oc", args...)
			path := filepath.Join("meta", "oc_"+strings.Join(args, "_"))
			return MarkErrorAsIgnorable(runner.Run(cmd, path))
		}
	}
	for _, p := range projects {
		p := p
		tasks <- func() error {
			cmd := exec.Command("oc", "-n", p, "policy", "can-i", "--list")
			path := filepath.Join("meta", "projects", p, "oc_-n_"+p+"_policy_can-i_--list")
			return MarkErrorAsIgnorable(runner.Run(cmd, path))
		}
	}
}
