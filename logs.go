package main

import (
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
func GetFetchLogsTasks(tasks chan<- Task, runner Runner, projects, resources []string) {
	loggableResources, err := GetLogabbleResources(projects, resources)
	if err != nil {
		tasks <- NewError(err)
		// continue and iterate over loggableResources even if there was
		// an error.
	}
	for _, r := range loggableResources {
		// Send task to fetch current logs.
		tasks <- FetchLogs(runner, r, *maxLogLines)
		// Send task to fetch previous logs.
		tasks <- FetchPreviousLogs(runner, r, *maxLogLines)
	}
}

// GetLogabbleResources returns a list of loggable resources. It may return
// results even in the presence of an error.
func GetLogabbleResources(projects, resources []string) ([]LoggableResource, error) {
	var (
		loggableResources []LoggableResource
		errors            errorList
	)
	for _, p := range projects {
		for _, rtype := range resources {
			names, err := GetResourceNames(p, rtype)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			for _, name := range names {
				resources, err := GetLoggableResources(p, rtype, name)
				if err != nil {
					errors = append(errors, err)
					continue
				}
				loggableResources = append(loggableResources, resources...)
			}
		}
	}
	if len(errors) > 0 {
		return loggableResources, errors
	}
	return loggableResources, nil
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
		return r.Run(cmd, path)
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
