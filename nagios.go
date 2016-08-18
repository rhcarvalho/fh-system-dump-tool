package main

import (
	"os/exec"
	"path/filepath"
)

// GetNagiosStatusData is a task factory for tasks that fetch Nagios status from
// the given pod in project.
func GetNagiosStatusData(r Runner, project, pod string) Task {
	return func() error {
		cmd := exec.Command("oc", "exec", pod, "--", "cat", "/var/log/nagios/status.dat")
		fname := pod + "_status.dat"
		path := filepath.Join("projects", project, "nagios", fname)
		return r.Run(cmd, path)
	}
}

// GetNagiosHistoricalData is a task factory for tasks that fetch Nagios
// archives from the given pod in project.
func GetNagiosHistoricalData(r Runner, project, pod string) Task {
	return func() error {
		cmd := exec.Command("oc", "exec", pod, "--", "tar", "-c", "-C", "/var/log/nagios", "archives")
		fname := pod + "_history.tar"
		path := filepath.Join("projects", project, "nagios", fname)
		return r.Run(cmd, path)
	}
}
