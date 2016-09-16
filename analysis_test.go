package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/feedhenry/fh-system-dump-tool/openshift/api/types"
)

var (
	podOk = types.Pod{
		ObjectMeta: types.ObjectMeta{
			Name:      "mongodb-2-1-x66za",
			Namespace: "qe-3node-4-1",
		},
		Status: types.PodStatus{
			ContainerStatuses: []types.ContainerStatus{
				{
					Name:  "mongodb-service",
					State: types.ContainerState{},
				},
			},
		},
	}
	podWaiting = types.Pod{
		ObjectMeta: types.ObjectMeta{
			Name:      "mongodb-2-1-x66za",
			Namespace: "qe-3node-4-1",
		},
		Status: types.PodStatus{
			ContainerStatuses: []types.ContainerStatus{
				{
					Name: "mongodb-service",
					State: types.ContainerState{
						Waiting: &types.ContainerStateWaiting{
							Reason:  "ContainerCreating",
							Message: "Image: docker.io/rhmap/mongodb:centos-3.2-29 is ready, container is creating",
						},
					},
				},
			},
		},
	}
)

var (
	normalEvent = types.Event{
		Message: "Test message",
		Type:    "Normal",
	}
	warningEvent = types.Event{
		TypeMeta: types.TypeMeta{Kind: "Event"},
		InvolvedObject: types.ObjectReference{
			Kind:      "Pod",
			Namespace: "qe-3node-4-1",
			Name:      "mongodb-2-1-x66za",
		},
		Reason:  "FailedSync",
		Message: "Error syncing pod, skipping: API error (500): Unknown device 84d72d4cf06a1e292ba21c36c5c93638ccd3c4cfab8bf048bd434ff9cdf43722\n",
		Count:   94215,
		Type:    "Warning",
	}
)

var (
	dcZeroReplicas = types.DeploymentConfig{
		ObjectMeta: types.ObjectMeta{
			Name:      "fh-mbaas",
			Namespace: "qe-3node-4-1",
		},
		Spec: types.DeploymentConfigSpec{
			Replicas: 0,
		},
	}
)

func TestCheckEvents(t *testing.T) {
	tests := []struct {
		description string
		eventList   types.EventList
		want        CheckResult
	}{
		{
			description: "warning event",
			eventList: types.EventList{
				Items: []types.Event{warningEvent},
			},
			want: CheckResult{
				CheckName: "check event log for errors",
				Ok:        false,
				Message:   "errors detected in event log",
				Events:    []types.Event{warningEvent},
			},
		},
	}

	for _, tt := range tests {
		if got := CheckEvents(tt.eventList); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckEvents(eventList) = \n%#v, want \n%#v", tt.description, got, tt.want)
		}
	}
}

func TestCheckDeploymentConfigs(t *testing.T) {
	tests := []struct {
		description string
		dcList      types.DeploymentConfigList
		want        CheckResult
	}{
		{
			description: "deployment config with zero replicas",
			dcList: types.DeploymentConfigList{
				Items: []types.DeploymentConfig{dcZeroReplicas},
			},
			want: CheckResult{
				CheckName: "check number of replicas in deployment configs",
				Ok:        false,
				Message:   "one or more deployment configs has number of replicas set to 0",
				Info: []Info{
					{
						Name:      dcZeroReplicas.ObjectMeta.Name,
						Namespace: dcZeroReplicas.ObjectMeta.Namespace,
						Message:   "the replica parameter is set to 0, this should be greater than 0",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		if got := CheckDeploymentConfigs(tt.dcList); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckDeploymentConfigs(dcList) = \n%#v, want \n%#v", tt.description, got, tt.want)
		}
	}
}

func TestCheckPods(t *testing.T) {
	tests := []struct {
		description string
		podList     types.PodList
		want        CheckResult
	}{
		{
			description: "empty pod list",
			podList: types.PodList{
				Items: []types.Pod{},
			},
			want: CheckResult{
				CheckName: "check pods for containers in waiting state",
				Ok:        true,
				Message:   "this issue was not detected",
			},
		},
		{
			description: "pod without container in waiting state",
			podList: types.PodList{
				Items: []types.Pod{podOk},
			},
			want: CheckResult{
				CheckName: "check pods for containers in waiting state",
				Ok:        true,
				Message:   "this issue was not detected",
			},
		},
		{
			description: "pod with container in waiting state",
			podList: types.PodList{
				Items: []types.Pod{podWaiting},
			},
			want: CheckResult{
				CheckName: "check pods for containers in waiting state",
				Ok:        false,
				Message:   "one or more containers are in waiting state",
				Info: []Info{
					{
						Name:      podWaiting.Status.ContainerStatuses[0].Name,
						Namespace: podWaiting.ObjectMeta.Namespace,
						Message:   "container mongodb-service in pod mongodb-2-1-x66za is in waiting state",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		if got := CheckPods(tt.podList); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckPods(podList) = \n%#v, want \n%#v", tt.description, got, tt.want)
		}
	}
}

func TestCheckProjectTask(t *testing.T) {
	tests := []struct {
		project    string
		definition DefinitionLoader
		want       []CheckResult
	}{
		{
			project:    "rhmap-project",
			definition: fakeDefinitionLoader{},
			want: []CheckResult{
				{
					CheckName: "check event log for errors",
					Ok:        true,
					Message:   "this issue was not detected",
				},
				{
					CheckName: "check number of replicas in deployment configs",
					Ok:        true,
					Message:   "this issue was not detected",
				},
				{
					CheckName: "check pods for containers in waiting state",
					Ok:        true,
					Message:   "this issue was not detected",
				},
			},
		},
		{
			project: "bad-project",
			definition: fakeDefinitionLoader{
				"events": types.EventList{
					Items: []types.Event{normalEvent, warningEvent},
				},
				"deploymentconfigs": types.DeploymentConfigList{
					Items: []types.DeploymentConfig{dcZeroReplicas},
				},
				"pods": types.PodList{
					Items: []types.Pod{podOk, podWaiting},
				},
			},
			want: []CheckResult{
				{
					CheckName: "check event log for errors",
					Ok:        false,
					Message:   "errors detected in event log",
					Events:    []types.Event{warningEvent},
				},
				{
					CheckName: "check number of replicas in deployment configs",
					Ok:        false,
					Message:   "one or more deployment configs has number of replicas set to 0",
					Info: []Info{
						{
							Name:      dcZeroReplicas.ObjectMeta.Name,
							Namespace: dcZeroReplicas.ObjectMeta.Namespace,
							Message:   "the replica parameter is set to 0, this should be greater than 0",
						},
					},
				},
				{
					CheckName: "check pods for containers in waiting state",
					Ok:        false,
					Message:   "one or more containers are in waiting state",
					Info: []Info{
						{
							Name:      podWaiting.Status.ContainerStatuses[0].Name,
							Namespace: podWaiting.ObjectMeta.Namespace,
							Message: fmt.Sprintf("container %s in pod %s is in waiting state",
								podWaiting.Status.ContainerStatuses[0].Name, podWaiting.ObjectMeta.Name),
						},
					},
				},
			},
		},
	}
	for i, tt := range tests {
		results := make(chan AnalysisResult, 1)
		task := CheckProjectTask(tt.project, tt.definition, results)

		if err := task(); err != nil {
			t.Errorf("%d: task() = %v, want %v", i, err, nil)
		}

		want := AnalysisResult{
			Projects: []ProjectResult{
				{
					Project: tt.project,
					Results: tt.want,
				},
			},
		}
		if result := <-results; !reflect.DeepEqual(result, want) {
			t.Errorf("%d: result = \n%#v, want \n%#v", i, result, want)
		}
	}
}

type fakeDefinitionLoader map[string]interface{}

func (l fakeDefinitionLoader) Load(kind string, v interface{}) {
	val := map[string]interface{}(l)[kind]
	b, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		panic(err)
	}
}

func (l fakeDefinitionLoader) Err() error {
	return nil
}

var _ DefinitionLoader = (*fakeDefinitionLoader)(nil)

func TestPrintAnalysisReport(t *testing.T) {
	tests := []struct {
		description    string
		analysisResult AnalysisResult
		// set want for exact matches or contains/notContains for
		// inexact matches.
		want        string
		contains    []string
		notContains []string
	}{
		{
			description:    "empty analysis",
			analysisResult: AnalysisResult{},
			want:           "No issues found\n",
		},
		{
			description: "no errors",
			analysisResult: AnalysisResult{
				Projects: []ProjectResult{
					{
						Project: "dev",
						Results: []CheckResult{
							{
								CheckName: "check event log for errors",
								Ok:        true,
								Message:   "this issue was not detected",
							},
							{
								CheckName: "check number of replicas in deployment configs",
								Ok:        true,
								Message:   "this issue was not detected",
							},
							{
								CheckName: "check pods for containers in waiting state",
								Ok:        true,
								Message:   "this issue was not detected",
							},
						},
					},
				},
			},
			want: "No issues found\n",
		},
		{
			description: "errors found",
			analysisResult: AnalysisResult{
				Projects: []ProjectResult{
					{
						Project: "rhmap-core",
						Results: []CheckResult{
							{
								CheckName: "check event log for errors",
								Ok:        false,
								Message:   "errors detected in event log",
								Events: []types.Event{
									{
										TypeMeta: types.TypeMeta{Kind: "Event"},
										InvolvedObject: types.ObjectReference{
											Namespace: "rhmap-core",
											Name:      "fh-ngui",
										},
										Reason:  "FailedUpdate",
										Message: "Cannot update deployment rhmap-core/fh-ngui-3 status to Pending: replicationcontrollers \"fh-ngui-3\" cannot be updated: the object has been modified; please apply your changes to the latest version and try again",
										Count:   1,
										Type:    "Warning",
									},
								},
							},
						},
					},
				},
			},
			contains:    []string{"rhmap-core", "fh-ngui", "Cannot update deployment"},
			notContains: []string{"No issues found"},
		},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		PrintAnalysisReport(tt.analysisResult, &out)
		got := out.String()
		if len(tt.contains) > 0 || len(tt.notContains) > 0 {
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("%s: got %q, want to contain %q", tt.description, got, want)
				}
			}
			for _, notWant := range tt.notContains {
				if strings.Contains(got, notWant) {
					t.Errorf("%s: got %q, want not to contain %q", tt.description, got, notWant)
				}
			}
		} else {
			if got != tt.want {
				t.Errorf("%s: got %q, want %q", tt.description, got, tt.want)
			}
		}
	}
}
