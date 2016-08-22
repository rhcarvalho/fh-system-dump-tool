package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Info struct {
	Name      string
	Namespace string
	Kind      string
	Count     int
	Message   string
}

type Result struct {
	CheckName     string `json:"checkName" yaml:"checkName"`
	Status        int    `json:"status" yaml:"status"`
	StatusMessage string `json:"statusMessage" yaml:"statusMessage"`
	Info          []Info `json:"info" yaml:"info"`
}

type Events struct {
	Items []struct {
		Kind           string `json:"kind"`
		InvolvedObject struct {
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		} `json:"involvedObject"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
		Count   int    `json:"count"`
	} `json:"items"`
}

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

type CheckTask func(string, string) (Result, error)

func GetAnalysisTasks(tasks chan<- Task, basepath string, projects []string, results chan<- CheckResults) {
	// Platform-wide analysis goes here

	// project specific analysis in here
	for _, p := range projects {
		tasks <- CheckProjectTask(p, basepath, results)
	}
}

// CheckTasks is a task factory for tasks that diagnose system conditions.
func CheckProjectTask(project, basepath string, results chan<- CheckResults) Task {
	return checkProjectTask(func() []CheckTask {
		return []CheckTask{CheckImagePullBackOff, CheckDeployConfigsReplicasNotZero}
	}, project, basepath, results)
}

// A getProjectCheckFactory generates tasks to diagnose system conditions.
type getProjectCheckFactory func() []CheckTask

type CheckResults struct {
	Scope   string `json:"scope"`
	Results []Result
}

// checkTasks executes all the CheckTasks returned from the supplied
// checkFactory against the specified project. The results of the checks are
// combined into a single JSON object and returned
func checkProjectTask(checkFactory getProjectCheckFactory, project, basepath string, results chan<- CheckResults) Task {
	return func() error {
		result := CheckResults{Scope: project, Results: []Result{}}
		checks := checkFactory()

		var errors errorList
		for _, check := range checks {
			res, err := check(project, basepath)
			if err != nil {
				errors = append(errors, err)
			}
			result.Results = append(result.Results, res)

		}

		results <- result

		if len(errors) > 0 {
			return errors
		}
		return nil
	}
}

// getDumpedResourceAsStruct retrieves the requested resource from the dump directory and
// decodes the JSON into the provided interface
func getDumpedResource(path []string, dest interface{}) error {
	filePath := filepath.Join(path...)
	contents, err := ioutil.ReadFile(filePath)

	decoder := json.NewDecoder(bytes.NewBuffer(contents))
	err = decoder.Decode(&dest)
	if err != nil {
		return err
	}

	return nil
}

// CheckImagePullBackOff checks all events in the supplied project and if any
// are exhibiting signs that they have experienced an ImagePullBackOff recently
// this will be reflected in the returned Result data. Any errors are written to
// the supplied stdErr writer.
func CheckImagePullBackOff(project, basepath string) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check deploys for ImagePullBackOff error"}
	events := Events{}
	err := getDumpedResource([]string{basepath, "projects", project, "definitions", "events.json"}, &events)
	if err != nil {
		result.Status = 2
		result.StatusMessage = "Error executing task"
		return result, err
	}

	for _, event := range events.Items {
		if event.Reason == "FailedSync" && strings.Contains(event.Message, "ImagePullBackOff") {
			info := Info{Name: event.InvolvedObject.Name, Namespace: event.InvolvedObject.Namespace, Kind: event.Kind, Count: event.Count, Message: event.Message}
			result.Status = 1
			result.StatusMessage = "'ImagePullBackOff' error detected"
			result.Info = append(result.Info, info)
		}
	}

	return result, nil
}

// CheckDeployConfigsReplicasNotZero checks all deployment configs in the
// supplied project and if any have replicas set to zero this will be reflected
// in the returned Result data. Any errors are written to the supplied stdErr
// writer.
func CheckDeployConfigsReplicasNotZero(project, basepath string) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check deployconfig replicas not 0"}
	deploymentConfigs := DeploymentConfigs{}
	err := getDumpedResource([]string{basepath, "projects", project, "definitions", "deploymentconfigs.json"}, &deploymentConfigs)
	if err != nil {
		result.Status = 2
		result.StatusMessage = "Error executing task"
		return result, err
	}

	for _, deploymentConfig := range deploymentConfigs.Items {
		if deploymentConfig.Spec.Replicas == 0 {
			info := Info{Name: deploymentConfig.Metadata.Name, Namespace: deploymentConfig.Metadata.Namespace, Kind: deploymentConfig.Kind, Count: 1, Message: "the replica parameter is set to 0, this should be greater than 0"}
			result.Status = 1
			result.StatusMessage = "one or more deployConfig replicas are set to 0"
			result.Info = append(result.Info, info)
		}
	}

	return result, nil
}
