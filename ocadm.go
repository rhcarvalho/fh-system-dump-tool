package main

import "os/exec"

// GetOcAdmDiagnosticsTasks sends tasks to fetch the oc adm diagnostics result.
func GetOcAdmDiagnosticsTask(runner Runner) Task {
	return func() error {
		cmd := exec.Command("oc", "adm", "diagnostics")
		path := "oc_adm_diagnostics"
		return runner.Run(cmd, path)
	}
}
