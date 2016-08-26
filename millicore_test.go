package main

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetMillicoreConfigTasks(t *testing.T) {
	tasks := make(chan Task, 1)
	runner := &FakeRunner{}

	GetMillicoreConfigTasks(tasks, runner, []string{"project1"}, func(project, resource, substr string) ([]string, error) {
		return []string{"millicore-1"}, nil
	})

	task := <-tasks
	err := task()
	if err != nil {
		t.Fatal(err)
	}
	expectedCalls := []RunCall{
		{
			[]string{"oc", "-n", "project1", "exec", "millicore-1", "--", "cat", "/etc/feedhenry/cluster-override.properties"},
			filepath.Join("projects", "project1", "millicore", "millicore-1_cluster-override.properties"),
		},
	}

	if !reflect.DeepEqual(runner.Calls, expectedCalls) {
		t.Errorf("runner.Calls = %q, want %q", runner.Calls, expectedCalls)
	}

}

func TestMillicorePodError(t *testing.T) {
	tasks := make(chan Task, 1)
	runner := &FakeRunner{}

	GetMillicoreConfigTasks(tasks, runner, []string{"project1"}, func(project, resource, substr string) ([]string, error) {
		return nil, errors.New("error retrieving pods")
	})

	task := <-tasks
	err := task()
	if err.Error() != "error retrieving pods" {
		t.Fatalf("Task(): Want: error retrieving pods, Got: " + err.Error())
	}
}
