package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

type Info struct {
	Name      string
	Namespace string
	Kind      string
	Count     int
	Message     string
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

type CheckTask func(string, io.Writer) (Result, error)

// ResourceDefinitions is a task factory for tasks that fetch the JSON resource
// definition for all given types in project. For each resource type, the task
// uses outFor and errOutFor to get io.Writers to write, respectively, the JSON
// output and any eventual error message.
func CheckTasks(project string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return checkTasks(func() []CheckTask {
		return []CheckTask{CheckImagePullBackOff, CheckDeployConfigsReplicasNotZero}
	}, project, outFor, errOutFor)
}

// A getProjectCheckFactory generates commands to get resources of a given
// type in a project.
type getProjectCheckFactory func() []CheckTask

type CheckResults struct {
	Results []Result
}

func checkTasks(checkFactory getProjectCheckFactory, project string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return func() error {
		stdOut, stdOutCloser, err := outFor(project, "analysis")
		if err != nil {
			return err
		}
		defer stdOutCloser.Close()
		stdErr, stdErrCloser, err := errOutFor(project, "analysis")
		if err != nil {
			return err
		}
		defer stdErrCloser.Close()

		results := CheckResults{Results: []Result{}}

		checks := checkFactory()

		var errors errorList
		for _, check := range checks {
			res, err := check(project, stdErr)
			if err != nil {
				errors = append(errors, err.Error())
			}
			results.Results = append(results.Results, res)

		}

		output, err := json.MarshalIndent(results, "", "    ")
		if err != nil {
			errors = append(errors, err.Error())
		}

		stdOut.Write(output)

		if len(errors) > 0 {
			return errors
		}
		return nil
	}
}

func getResourceReader(project, resource string) (io.Reader, error) {
	stdOut := bytes.NewBuffer([]byte{})
	stdErr := bytes.NewBuffer([]byte{})
	outFor := func(project, resource string) (io.Writer, io.Closer, error) {
		return stdOut, ioutil.NopCloser(nil), nil
	}
	errOutFor := func(project, resource string) (io.Writer, io.Closer, error) {
		return stdErr, ioutil.NopCloser(nil), nil
	}
	task := ResourceDefinitions(project, []string{resource}, outFor, errOutFor)

	err := task()
	if err != nil {
		return nil, err
	}

	stdErrString := string(stdErr.Bytes())
	if stdErrString != "" {
		return nil, errors.New(stdErrString)
	}

	return stdOut, nil
}

func CheckImagePullBackOff(project string, stdErr io.Writer) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check deploys for ImagePullBackOff error"}
	events := Events{}
	eventsJson, err := getResourceReader(project, "events")
	if err != nil {
		stdErr.Write([]byte(err.Error()))
		return result, err
	}
	result := Result{Status: 0, StatusMessage: "No issues detected", CheckName: "Check ImagePullBackOff"}
	events := Events{}

	decoder := json.NewDecoder(eventsJson)
	err = decoder.Decode(&events)
	if err != nil {
		stdErr.Write([]byte(err.Error()))
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

func CheckDeployConfigsReplicasNotZero(project string, stdErr io.Writer) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check deployconfig replicas not 0"}
	deploymentConfigs := DeploymentConfigs{}
	dcJson, err := getResourceReader(project, "dc")
	if err != nil {
		stdErr.Write([]byte(err.Error()))
		return result, err
	}

	decoder := json.NewDecoder(dcJson)
	err = decoder.Decode(&deploymentConfigs)
	if err != nil {
		stdErr.Write([]byte(err.Error()))
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
	output, err := json.MarshalIndent(result, "", "    ")

	stdOut.Write(output)

	return nil
}
