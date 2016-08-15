package main

import (
	"fmt"
	"os/exec"
)

// GetNagiosStatusData is a task factory for tasks that fetch a JSON representation of
// the provided project. The task uses outFor and errOutFor to get io.Writers
// to write, respectively, the JSON output and any eventual error message.
func GetNagiosStatusData(project, pod string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return retrieveNagiosData(func() *exec.Cmd {
		return exec.Command("oc", "exec", pod, "--", "cat", "/var/log/nagios/status.dat")
	}, project, pod, "status", outFor, errOutFor)
}

// GetNagiosHistoricalData is a task factory for tasks that fetch a JSON representation of
// the provided project. The task uses outFor and errOutFor to get io.Writers
// to write, respectively, the JSON output and any eventual error message.
func GetNagiosHistoricalData(project, pod string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return retrieveNagiosData(func() *exec.Cmd {
		return exec.Command("oc", "exec", pod, "--", "tar", "-c", "-C", "/var/log/nagios", "archives")
	}, project, pod, "history", outFor, errOutFor)
}

// A getNagiosDataCmdFactory generates commands to get the raw nagios status data
// in the provided project from the provided pod
type getNagiosDataCmdFactory func() *exec.Cmd

// retrieveNagiosData will dump a JSON representation of the current nagios status into
// the io.Writer returned from outFor and log any errors it encounters doing so
// into the io.Writer returned from errOutFor.
func retrieveNagiosData(cmdFactory getNagiosDataCmdFactory, project, pod, nagiosResourceType string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return func() error {
		stdout, stdoutCloser, err := outFor(project, pod+"-"+nagiosResourceType)
		if err != nil {
			return err
		}
		defer stdoutCloser.Close()

		stderr, stderrCloser, err := errOutFor(project, pod+"-"+nagiosResourceType)
		if err != nil {
			return err
		}
		defer stderrCloser.Close()

		cmd := cmdFactory()

		if err := runCmdCaptureOutput(cmd, stdout, stderr); err != nil {
			fmt.Fprint(stderr, err)
			return err
		}
		return nil
	}
}
