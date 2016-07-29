package main

import (
	"io"
	"os/exec"
	"strconv"
)

// LoggableResource describes an OpenShift resource that produces logs.
type LoggableResource struct {
	Project string
	// Type should be one of: build, buildconfig, deploymentconfig or pod,
	// or an alias to one of those.
	Type string
	Name string
	// Container is required for pods with more than one container, and
	// ignored for other types.
	Container string
}

// FetchLogs is a task factory for tasks that fetch the logs of a
// LoggableResource. Set maxLines to limit how many lines are fetched. Logs are
// written to out and eventual error messages go to errOut.
func FetchLogs(resource LoggableResource, maxLines int, out, errOut io.Writer) Task {
	return ocLogs(resource, maxLines, nil, out, errOut)
}

// FetchPreviousLogs is like FetchLogs, but for the previous version of a
// resource.
func FetchPreviousLogs(resource LoggableResource, maxLines int, out, errOut io.Writer) Task {
	return ocLogs(resource, maxLines, []string{"--previous"}, out, errOut)
}

// ocLogs fetches logs from OpenShift resources using oc.
func ocLogs(resource LoggableResource, maxLines int, extraArgs []string, out, errOut io.Writer) Task {
	return fetchLogs(func(resource LoggableResource) *exec.Cmd {
		return exec.Command("oc", append([]string{
			"-n", resource.Project,
			"logs", resource.Type + "/" + resource.Name,
			"-c", resource.Container,
			"--tail", strconv.Itoa(maxLines)},
			extraArgs...)...)
	}, resource, out, errOut)
}

// A logsCmdFactory generates commands to fetch logs of a given resource type
// and name.
type logsCmdFactory func(resource LoggableResource) *exec.Cmd

func fetchLogs(cmdFactory logsCmdFactory, resource LoggableResource, out, errOut io.Writer) Task {
	return func() error {
		cmd := cmdFactory(resource)
		return runCmdCaptureOutput(cmd, out, errOut)
	}
}

// GetLoggableResources returns a list of loggable resources for the named
// resource of type rtype in the given project. Only pods may return multiple
// loggable resources, as many as the number of containers in the pod.
func GetLoggableResources(project, rtype, name string) ([]LoggableResource, error) {
	return getLoggableResources(GetPodContainers, project, rtype, name)
}

func getLoggableResources(getPodContainers func(string, string) ([]string, error), project, rtype, name string) ([]LoggableResource, error) {
	var (
		loggableResources []LoggableResource
		containers        []string
	)
	switch rtype {
	case "po", "pod", "pods":
		var err error
		containers, err = getPodContainers(project, name)
		if err != nil {
			return nil, err
		}
	default:
		// For types other than pod, we can treat them as if
		// they had a single unnamed container, for the name
		// doesn't matter when fetching logs.
		containers = []string{""}
	}
	for _, container := range containers {
		loggableResources = append(loggableResources,
			LoggableResource{
				Project:   project,
				Type:      rtype,
				Name:      name,
				Container: container,
			})
	}
	return loggableResources, nil
}

// GetPodContainers returns a list of container names for the named pod in the
// project.
func GetPodContainers(project, name string) ([]string, error) {
	return getSpaceSeparated(exec.Command("oc", "-n", project, "get", "pod", name, "-o=jsonpath={.spec.containers[*].name}"))
}
