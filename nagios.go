package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetNagiosTasks sends tasks to dump Nagios data for each project that contain
// a Nagios pod. It is an error if no projects contain a Nagios pod.
func GetNagiosTasks(tasks chan<- Task, runner Runner, projects []string) {
	foundANagiosPod := false
	for _, p := range projects {
		pods, err := getResourceNamesBySubstr(p, "pod", "nagios")
		if err != nil {
			tasks <- NewError(err)
			continue
		}
		for _, pod := range pods {
			foundANagiosPod = true
			tasks <- GetNagiosStatusData(runner, p, pod)
			tasks <- GetNagiosHistoricalData(runner, p, pod)
		}
	}
	if !foundANagiosPod {
		tasks <- NewError(errors.New("A Nagios pod could not be found in any project. For a more thorough analysis, please ensure Nagios is running in all RHMAP projects."))
	}
}

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

type ResourceMatchFactory func(project, resource, substr string) ([]string, error)

// getResourceNamesBySubstr returns a list of names for the provided resource type that contain
// the provided string, in the provided project.
func getResourceNamesBySubstr(project, resource, substr string) ([]string, error) {
	// FIXME: take runner as argument.
	runner := simpleRunner{}
	resources, err := GetResourceNames(runner, project, resource)
	if err != nil {
		return nil, err
	}

	filtered := resources[:0]

	for _, resource := range resources {
		if strings.Contains(resource, substr) {
			filtered = append(filtered, resource)
		}
	}

	return filtered, nil
}
