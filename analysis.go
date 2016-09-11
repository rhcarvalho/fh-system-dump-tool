package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

// A CheckResult is the result of some verification of the system conditions.
type CheckResult struct {
	CheckName string  `json:"name"`
	Ok        bool    `json:"ok"`
	Message   string  `json:"message"`
	Info      []Info  `json:"info,omitempty"`
	Events    []Event `json:"events,omitempty"`
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

// Some of the types below are taken from:
// https://github.com/openshift/origin/blob/master/vendor/k8s.io/kubernetes/pkg/api/types.go
// However the fields that were not required for our purposes were removed for brevity.
// Currently the copied definitions are:
// - ContainerStateWaiting

// Event is a representation of the items in the OpenShift event log, this is
// trimmed to only the required fields.
type Event struct {
	Kind           string `json:"kind"`
	InvolvedObject struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	} `json:"involvedObject"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Type    string `json:"type"`
}

// Events is a representation of everything in the OpenShift event log for a
// particular project.
type Events struct {
	Items []Event `json:"items"`
}

// DeploymentConfigs is a representation of the OpenShift deployment configs.
type DeploymentConfigs struct {
	Items []struct {
		Kind     string `json:"kind"`
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
		} `json:"spec"`
	} `json:"items"`
}

// ContainerStateWaiting is one possible status that a container can be in.
type ContainerStateWaiting struct {
	// A brief CamelCase string indicating details about why the container is in waiting state.
	Reason string `json:"reason,omitempty"`
	// A human-readable message indicating details about why the container is in waiting state.
	Message string `json:"message,omitempty"`
}

// Pods is a representation of all the pods in a project from OpenShift.
type Pods struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Status struct {
			ContainerStatuses []struct {
				Name  string `json:"name"`
				State struct {
					Waiting *ContainerStateWaiting `json:"waiting,omitempty"`
				} `json:"state"`
			} `json:"containerStatuses"`
		} `json:"status"`
	} `json:"items"`
}

// CheckTask is the interface which checks must implement.
type CheckTask func(DumpedJSONResourceFactory) (CheckResult, error)

// GetAnalysisTasks creates all the analysis tasks and sends them one by one
// down the tasks Channel.
func GetAnalysisTasks(tasks chan<- Task, basepath string, projects []string, results chan<- AnalysisResult) {
	// Platform-wide analysis goes here.

	// Project-specific analysis goes here.
	for _, p := range projects {
		JSONResourceFactory := getDumpedJSONResourceFactory(filepath.Join(basepath, "projects", p))
		tasks <- CheckProjectTask(p, results, JSONResourceFactory)
	}
}

// CheckProjectTask is a task factory for tasks that diagnose system conditions.
func CheckProjectTask(project string, results chan<- AnalysisResult, JSONResourceFactory DumpedJSONResourceFactory) Task {
	return checkProjectTask(func() []CheckTask {
		return []CheckTask{CheckEventLogForErrors, CheckDeployConfigsReplicasNotZero, CheckForWaitingPods}
	}, JSONResourceFactory, project, results)
}

// A getProjectCheckFactory generates tasks to diagnose system conditions.
type getProjectCheckFactory func() []CheckTask

// checkTasks executes all the CheckTasks returned from the supplied
// checkFactory against the specified project. The results of the checks are
// combined into a single JSON object and returned.
func checkProjectTask(checkFactory getProjectCheckFactory, JSONResourceFactory DumpedJSONResourceFactory, project string, results chan<- AnalysisResult) Task {
	return func() error {
		result := ProjectResult{Project: project}
		checks := checkFactory()

		var errors errorList
		for _, check := range checks {
			res, err := check(JSONResourceFactory)
			if err != nil {
				errors = append(errors, err)
			}
			result.Results = append(result.Results, res)

		}

		results <- AnalysisResult{Projects: []ProjectResult{result}}

		if len(errors) > 0 {
			return errors
		}
		return nil
	}
}

// DumpedJSONResourceFactory takes a string containing the path to a
// JSON file which will be parsed and loaded into the supplied interface.
type DumpedJSONResourceFactory func(string, interface{}) error

// getDumpedJSONResourceFactory returns a factory which will load resource from
// a given basepath. The factory parses the file contents as JSON and loads it
// into the provided dest interface.
func getDumpedJSONResourceFactory(basepath string) DumpedJSONResourceFactory {
	return func(path string, dest interface{}) error {
		contents, err := ioutil.ReadFile(filepath.Join(basepath, path))
		if err != nil {
			return err
		}
		return json.Unmarshal(contents, dest)
	}
}

// CheckForWaitingPods checks all pods for any containers in waiting status.
func CheckForWaitingPods(JSONResourceFactory DumpedJSONResourceFactory) (CheckResult, error) {
	result := CheckResult{
		CheckName: "check pods for containers in waiting state",
		Ok:        true,
		Message:   "this issue was not detected",
	}
	var pods Pods
	if err := JSONResourceFactory(filepath.Join("definitions", "pods.json"), &pods); err != nil {
		result.Ok = false
		result.Message = "Error executing task: " + err.Error()
		return result, err
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Waiting != nil {
				result.Ok = false
				result.Message = "one or more containers are in waiting state"
				msg := "container " + container.Name + " in pod " + pod.Metadata.Name + " is in waiting state"
				info := Info{Name: container.Name, Namespace: pod.Metadata.Namespace, Message: msg}
				result.Info = append(result.Info, info)
			}
		}
	}

	return result, nil
}

// CheckEventLogForErrors checks all events in the supplied project and if any
// are not type 'Normal' (i.e. Warning or Error), it will add them to the
// returned results.
func CheckEventLogForErrors(JSONResourceFactory DumpedJSONResourceFactory) (CheckResult, error) {
	result := CheckResult{
		CheckName: "check event log for errors",
		Ok:        true,
		Message:   "this issue was not detected",
	}
	var events Events
	if err := JSONResourceFactory(filepath.Join("definitions", "events.json"), &events); err != nil {
		result.Ok = false
		result.Message = "Error executing task: " + err.Error()
		return result, err
	}

	for _, event := range events.Items {
		if event.Type != "Normal" {
			result.Ok = false
			result.Message = "Errors detected in event log"
			result.Events = append(result.Events, event)
		}
	}

	return result, nil
}

// CheckDeployConfigsReplicasNotZero checks all deployment configs in the
// supplied JSON Resource Factory, and if any are found with a replica of 0, it
// will add a note about it to the returned result.
func CheckDeployConfigsReplicasNotZero(ResourceFactory DumpedJSONResourceFactory) (CheckResult, error) {
	result := CheckResult{
		CheckName: "check number of replicas in deployment configs",
		Ok:        true,
		Message:   "this issue was not detected",
	}
	var deploymentConfigs DeploymentConfigs
	err := ResourceFactory(filepath.Join("definitions", "deploymentconfigs.json"), &deploymentConfigs)
	if err != nil {
		result.Ok = false
		result.Message = "Error executing task: " + err.Error()
		return result, err
	}

	for _, deploymentConfig := range deploymentConfigs.Items {
		if deploymentConfig.Spec.Replicas == 0 {
			info := Info{Name: deploymentConfig.Metadata.Name, Namespace: deploymentConfig.Metadata.Namespace, Message: "the replica parameter is set to 0, this should be greater than 0"}
			result.Ok = false
			result.Message = "one or more deployConfig replicas are set to 0"
			result.Info = append(result.Info, info)
		}
	}

	return result, nil
}
