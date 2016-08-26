package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Info struct {
	Name      string
	Namespace string
	Kind      string
	Count     int
	Message   string
}

type Result struct {
	CheckName     string  `json:"checkName" yaml:"checkName"`
	Status        int     `json:"status" yaml:"status"`
	StatusMessage string  `json:"statusMessage" yaml:"statusMessage"`
	Info          []Info  `json:"info" yaml:"info"`
	Events        []Event `json:"events" yaml:"events"`
}

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

type Events struct {
	Items []Event `json:"items"`
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

type ContainerWaiting struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

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
					Waiting *ContainerWaiting `json:"waiting,omitempty"`
				} `json:"state"`
			} `json:"containerStatuses"`
		} `json:"status"`
	} `json:"items"`
}

type CheckTask func(DumpedJSONResourceFactory) (Result, error)

func GetAnalysisTasks(tasks chan<- Task, basepath string, projects []string, results chan<- CheckResults) {
	// Platform-wide analysis goes here

	// project specific analysis in here
	for _, p := range projects {
		JSONResourceFactory := getDumpedJSONResourceFactory([]string{basepath, "projects", p})
		tasks <- CheckProjectTask(p, results, JSONResourceFactory)
	}
}

// CheckTasks is a task factory for tasks that diagnose system conditions.
func CheckProjectTask(project string, results chan<- CheckResults, JSONResourceFactory DumpedJSONResourceFactory) Task {
	return checkProjectTask(func() []CheckTask {
		return []CheckTask{CheckEventLogForErrors, CheckDeployConfigsReplicasNotZero, CheckForWaitingPods}
	}, JSONResourceFactory, project, results)
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
func checkProjectTask(checkFactory getProjectCheckFactory, JSONResourceFactory DumpedJSONResourceFactory, project string, results chan<- CheckResults) Task {
	return func() error {
		result := CheckResults{Scope: project, Results: []Result{}}
		checks := checkFactory()

		var errors errorList
		for _, check := range checks {
			res, err := check(JSONResourceFactory)
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

// Takes an array of strings describing the path to a JSON file
// which will be parsed and loaded into the supplied interface
type DumpedJSONResourceFactory func([]string, interface{}) error

// Returns a factory which will load resource from a given basepath. The factory parses the file
// contents as JSON and loads it into the provided dest interface
func getDumpedJSONResourceFactory(basepath []string) DumpedJSONResourceFactory {

	return func(path []string, dest interface{}) error {
		file := filepath.Join(append(basepath, path...)...)
		contents, err := os.Open(file)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(contents)
		if err := decoder.Decode(&dest); err != nil {
			return err
		}
		return nil
	}

}

// CheckForWaitingPods checks all pods for any containers in waiting status
func CheckForWaitingPods(JSONResourceFactory DumpedJSONResourceFactory) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check pods for 'waiting' containers", Info: []Info{}, Events: []Event{}}
	pods := Pods{}
	if err := JSONResourceFactory([]string{"definitions", "pods.json"}, &pods); err != nil {
		result.Status = 2
		result.StatusMessage = "Error executing task: " + err.Error()
		return result, err
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Waiting != nil {
				result.Status = 1
				result.StatusMessage = "Waiting containers have been detected"
				msg := "container " + container.Name + " in pod " + pod.Metadata.Name + " is in waiting state"
				info := Info{Name: container.Name, Count: 1, Namespace: pod.Metadata.Namespace, Kind: "container", Message: msg}
				result.Info = append(result.Info, info)
			}
		}
	}

	return result, nil
}

// CheckEventLogForErrors checks all events in the supplied project and if any
// are not type 'Normal' (i.e. Warning or Error), it will add them to the returned results.
func CheckEventLogForErrors(JSONResourceFactory DumpedJSONResourceFactory) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check eventlog for any errors", Info: []Info{}, Events: []Event{}}
	events := Events{}
	if err := JSONResourceFactory([]string{"definitions", "events.json"}, &events); err != nil {
		result.Status = 2
		result.StatusMessage = "Error executing task"
		return result, err
	}

	for _, event := range events.Items {
		if event.Type != "Normal" {
			result.Status = 1
			result.StatusMessage = "Errors detected in event log"
			result.Events = append(result.Events, event)
		}
	}

	return result, nil
}

// CheckDeployConfigsReplicasNotZero checks all deployment configs in the supplied
// JSON Resource Factory, and if any are found with a replica of 0, it will add a
// note about it to the returned result
func CheckDeployConfigsReplicasNotZero(ResourceFactory DumpedJSONResourceFactory) (Result, error) {
	result := Result{Status: 0, StatusMessage: "this issue was not detected", CheckName: "check deployconfig replicas not 0", Info: []Info{}, Events: []Event{}}
	deploymentConfigs := DeploymentConfigs{}
	err := ResourceFactory([]string{"definitions", "deploymentconfigs.json"}, &deploymentConfigs)
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
