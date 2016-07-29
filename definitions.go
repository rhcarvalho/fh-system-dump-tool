package main

import "os/exec"

// ResourceDefinitions is a task factory for tasks that fetch the JSON resource
// definition for all given types in project. For each resource type, the task
// uses outFor and errOutFor to get io.Writers to write, respectively, the JSON
// output and any eventual error message.
func ResourceDefinitions(project string, types []string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return resourceDefinitions(func(project, resource string) *exec.Cmd {
		return exec.Command("oc", "-n", project, "get", resource, "-o=json")
	}, project, types, outFor, errOutFor)
}

// A getProjectResourceCmdFactory generates commands to get resources of a given
// type in a project.
type getProjectResourceCmdFactory func(project, resource string) *exec.Cmd

func resourceDefinitions(cmdFactory getProjectResourceCmdFactory, project string, types []string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return func() error {
		var errors errorList
		// NOTE: we could fetch all resources of all types in a single
		// call to oc, by passing a comma-separated list of resource
		// types. Instead, we call oc multiple times to send the output
		// to different files without processing the contents of the
		// output from oc.
		for _, resource := range types {
			cmd := cmdFactory(project, resource)
			if err := runCmdCaptureOutput(cmd, project, resource, outFor, errOutFor); err != nil {
				// In case of errors, report it, skip the
				// current resource type and proceed with the
				// next.
				errors = append(errors, err.Error())
				continue
			}
		}
		if len(errors) > 0 {
			return errors
		}
		return nil
	}
}
