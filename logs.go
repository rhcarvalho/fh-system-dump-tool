package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strconv"
)

// LoggableResource describes an OpenShift resource that produces logs. Even
// though oc logs can fetch logs for build, buildconfig, deploymentconfig and
// pod resources, eventually the first three are just shortcuts to a certain
// pod. For dump purposes, it is enough to fetch logs of all pods.
type LoggableResource struct {
	Project string
	// Type should be one of: build, buildconfig, deploymentconfig or pod,
	// or an alias to one of those.
	Type string
	// Name is generally a pod name, but could be a reference to one of the
	// other types understood by oc logs.
	Name string
	// Container is required for pods with more than one container.
	Container string
}

// GetFetchLogsTasks sends tasks to fetch current and previous logs of all
// resources in all projects.
func GetFetchLogsTasks(tasks chan<- Task, runner Runner, projects, resources []string, maxLines int) {
	for _, p := range projects {
		for _, rtype := range resources {
			names, err := GetResourceNames(runner, p, rtype)
			if err != nil {
				tasks <- NewError(err)
				continue
			}
			for _, name := range names {
				getFetchLogsTasksPerResource(tasks, runner, p, rtype, name, maxLines)
			}
		}
	}
}

// getFetchLogsTasksPerResource sends tasks to fetch current and previous logs
// of the named resource of type rtype in the given project. Pod resources
// produce tasks for each container in the pod.
func getFetchLogsTasksPerResource(tasks chan<- Task, runner Runner, project, rtype, name string, maxLines int) {
	var (
		containers []string
	)
	switch rtype {
	case "po", "pod", "pods":
		var err error
		containers, err = GetPodContainers(runner, project, name)
		if err != nil {
			tasks <- NewError(err)
			return
		}
	default:
		// For types other than pod, we can treat them as if
		// they had a single unnamed container, for the name
		// doesn't matter when fetching logs.
		containers = []string{""}
	}
	for _, container := range containers {
		r := LoggableResource{
			Project:   project,
			Type:      rtype,
			Name:      name,
			Container: container,
		}
		// Send task to fetch current logs.
		tasks <- FetchLogs(runner, r, maxLines)
		// Send task to fetch previous logs.
		tasks <- FetchPreviousLogs(runner, r, maxLines)
	}
}

// GetPodContainers returns a list of container names for the named pod in the
// project.
func GetPodContainers(runner Runner, project, name string) ([]string, error) {
	cmd := exec.Command("oc", "-n", project, "get", "pod", name, "-o=jsonpath={.spec.containers[*].name}")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := runner.Run(cmd, filepath.Join("projects", project, "pods", name, "container-names")); err != nil {
		return nil, err
	}
	names, err := readSpaceSeparated(&b)
	return names, MarkErrorAsIgnorable(err)
}

// FetchLogs is a task factory for tasks that fetch the logs of a
// LoggableResource. Set maxLines to limit how many lines are fetched. Logs are
// written to out and eventual error messages go to errOut.
func FetchLogs(r Runner, resource LoggableResource, maxLines int) Task {
	return ocLogs(r, resource, maxLines, nil, "logs")
}

// FetchPreviousLogs is like FetchLogs, but for the previous version of a
// resource.
func FetchPreviousLogs(r Runner, resource LoggableResource, maxLines int) Task {
	return ocLogs(r, resource, maxLines, []string{"--previous"}, "logs-previous")
}

// ocLogs fetches logs from an OpenShift resource using oc.
func ocLogs(r Runner, resource LoggableResource, maxLines int, extraArgs []string, what string) Task {
	return func() error {
		name := resource.Name
		if resource.Type != "" {
			name = resource.Type + "/" + name
		}
		cmd := exec.Command("oc", append([]string{
			"-n", resource.Project,
			"logs", name,
			"-c", resource.Container,
			"--tail", strconv.Itoa(maxLines)},
			extraArgs...)...)
		filename := resource.Name
		if resource.Type != "" {
			filename = resource.Type + "_" + filename
		}
		if resource.Container != "" {
			filename += "_" + resource.Container
		}
		path := filepath.Join("projects", resource.Project, what, filename+".logs")
		return MarkErrorAsIgnorable(r.Run(cmd, path))
	}
}
