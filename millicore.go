package main

import (
	"os/exec"
	"path/filepath"
)

// GetMillicoreConfigTasks will create a function matching the CheckTask
// interface for each project to retrieve the Millicore config in that
// property, if the pod exists. The created tasks are sent down the tasks
// channel.
func GetMillicoreConfigTasks(tasks chan<- Task, runner Runner, projects []string, resourceFactory ResourceMatchFactory) {
	for _, p := range projects {
		pods, err := resourceFactory(p, "pod", "millicore")
		if err != nil {
			tasks <- NewError(err)
			continue
		}
		for _, pod := range pods {
			tasks <- GetMillicoreConfig(runner, p, pod)
		}
	}
}

// GetMillicoreConfig will retrieve the Millicore config from the Millicore
// container inside the provided pod and project.
func GetMillicoreConfig(r Runner, project, pod string) Task {
	return func() error {
		cmd := exec.Command("oc", "-n", project, "exec", pod, "--", "cat", "/etc/feedhenry/cluster-override.properties")
		path := filepath.Join("projects", project, "millicore", pod+"_cluster-override.properties")
		return r.Run(cmd, path)
	}
}
