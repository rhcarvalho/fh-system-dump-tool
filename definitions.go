package main

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// GetResourceDefinitionTasks sends tasks to fetch the definitions of all
// resources in all projects.
func GetResourceDefinitionTasks(tasks chan<- Task, runner Runner, projects, resources []string) {
	// NOTE: we could fetch all resources of all types in a single call to
	// oc, by passing a comma-separated list of resource types. Instead, we
	// call oc multiple times to send the output to different files without
	// processing the contents of the output from oc.
	for _, p := range projects {
		for _, resource := range resources {
			tasks <- ResourceDefinition(runner, p, resource)
		}
	}
}

// ResourceDefinition is a task factory for tasks that fetch the JSON resource
// definition for the given resource in the given project.
func ResourceDefinition(r Runner, project, resource string) Task {
	return func() error {
		var args []string
		if project != "" {
			args = append(args, "-n", project)
		}
		args = append(args, "get", resource, "-o=json")
		cmd := exec.Command("oc", args...)

		var tree []string
		if project != "" {
			tree = append(tree, "projects", project)
		}
		fname := strings.Replace(resource, "/", "_", -1) + ".json"
		tree = append(tree, "definitions", fname)

		path := filepath.Join(tree...)
		return r.Run(cmd, path)
	}
}

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
			if err := runCmdCaptureOutputDeprecated(cmd, project, resource, outFor, errOutFor); err != nil {
				// In case of errors, report it, skip the
				// current resource type and proceed with the
				// next.
				errors = append(errors, err)
				continue
			}
		}
		if len(errors) > 0 {
			return errors
		}
		return nil
	}
}
