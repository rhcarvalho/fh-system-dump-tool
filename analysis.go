package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/feedhenry/fh-system-dump-tool/openshift/api/types"
)

// A CheckResult is the result of some verification of the system conditions.
type CheckResult struct {
	CheckName string        `json:"name"`
	Ok        bool          `json:"ok"`
	Message   string        `json:"message"`
	Info      []Info        `json:"info,omitempty"`
	Events    []types.Event `json:"events,omitempty"`
}

// ProjectResult stores the results of checks in a project.
type ProjectResult struct {
	Project string        `json:"project"`
	Results []CheckResult `json:"checks"`
}

// AnalysisResult aggregates the result of checks executed against the system.
// It is used to dump analysis results to a JSON file.
type AnalysisResult struct {
	Platform []CheckResult   `json:"platform,omitempty"`
	Projects []ProjectResult `json:"projects,omitempty"`
}

// Info is a piece of information regarding a check, multiple Info can be
// attached to a single Result.
type Info struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Message   string `json:"message"`
}

// GetAnalysisTasks creates all the analysis tasks and sends them one by one
// down the tasks Channel.
func GetAnalysisTasks(tasks chan<- Task, basepath string, projects []string, results chan<- AnalysisResult) {
	// Platform-wide analysis goes here.

	// Project-specific analysis goes here.
	for _, p := range projects {
		definition := &definitionLoader{basepath: basepath, project: p}
		tasks <- CheckProjectTask(p, definition, results)
	}
}

// CheckProjectTask returns a task that diagnoses problems in the project scope.
func CheckProjectTask(project string, definition DefinitionLoader, results chan<- AnalysisResult) Task {
	return func() error {
		result := ProjectResult{Project: project}

		var (
			events            types.EventList
			deploymentConfigs types.DeploymentConfigList
			pods              types.PodList
		)

		definition.Load("events", &events)
		definition.Load("deploymentconfigs", &deploymentConfigs)
		definition.Load("pods", &pods)

		if err := definition.Err(); err != nil {
			return err
		}

		result.Results = append(result.Results,
			CheckEvents(events),
			CheckDeploymentConfigs(deploymentConfigs),
			CheckPods(pods),
		)

		results <- AnalysisResult{Projects: []ProjectResult{result}}

		return nil
	}
}

// A DefinitionLoader loads definitions of OpenShift resources.
type DefinitionLoader interface {
	Load(kind string, v interface{})
	Err() error
}

// definitionLoader loads JSON resource definitions from files.
type definitionLoader struct {
	basepath, project string
	err               error
}

func (l *definitionLoader) Load(kind string, v interface{}) {
	if l.err != nil {
		return
	}
	path := filepath.Join(l.basepath, "projects", l.project, "definitions", kind+".json")
	l.err = load(path, &v)
}

func (l *definitionLoader) Err() error {
	return l.err
}

// load loads JSON resource from a given path into dest.
func load(path string, dest interface{}) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(contents, dest)
}

// CheckPods checks all pods for any containers in waiting status.
func CheckPods(pods types.PodList) CheckResult {
	result := CheckResult{
		CheckName: "check pods for containers in waiting state",
		Ok:        true,
		Message:   "this issue was not detected",
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Waiting != nil {
				result.Ok = false
				result.Message = "one or more containers are in waiting state"
				result.Info = append(result.Info, Info{
					Name:      container.Name,
					Namespace: pod.ObjectMeta.Namespace,
					Message:   fmt.Sprintf("container %s in pod %s is in waiting state", container.Name, pod.ObjectMeta.Name),
				})
			}
		}
	}
	return result
}

// CheckEvents checks all events looking for events which type is not Normal
// (i.e., Warning or Error).
func CheckEvents(events types.EventList) CheckResult {
	result := CheckResult{
		CheckName: "check event log for errors",
		Ok:        true,
		Message:   "this issue was not detected",
	}
	for _, event := range events.Items {
		if event.Type != "Normal" {
			result.Ok = false
			result.Message = "errors detected in event log"
			result.Events = append(result.Events, event)
		}
	}
	return result
}

// CheckDeploymentConfigs checks that all deployment configs have a non-zero
// number of replicas configured.
func CheckDeploymentConfigs(deploymentConfigs types.DeploymentConfigList) CheckResult {
	result := CheckResult{
		CheckName: "check number of replicas in deployment configs",
		Ok:        true,
		Message:   "this issue was not detected",
	}
	for _, deploymentConfig := range deploymentConfigs.Items {
		if deploymentConfig.Spec.Replicas == 0 {
			result.Ok = false
			result.Message = "one or more deployment configs has number of replicas set to 0"
			result.Info = append(result.Info, Info{
				Name:      deploymentConfig.ObjectMeta.Name,
				Namespace: deploymentConfig.ObjectMeta.Namespace,
				Message:   "the replica parameter is set to 0, this should be greater than 0",
			})
		}
	}
	return result
}
