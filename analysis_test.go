package main

import (
	"errors"
	"testing"
)

func mockCheckFactoryOnePassOneFail() []CheckTask {
	return []CheckTask{mockTestOne, mockTestTwo}
}
func mockCheckFactoryOnePass() []CheckTask {
	return []CheckTask{mockTestOne}
}

func mockTestOne(project, basepath string) (Result, error) {
	result := Result{StatusMessage: "Called mockTestOne"}
	return result, nil
}

func mockTestTwo(project, basepath string) (Result, error) {
	result := Result{StatusMessage: "Called mockTestTwo"}
	return result, errors.New("FAIL")
}

func TestCheckTasksWithFail(t *testing.T) {
	results := make(chan CheckResults, 1)
	defer close(results)
	task := checkProjectTask(mockCheckFactoryOnePassOneFail, "MockProject", "/", results)

	err := task()

	if err == nil {
		t.Fatal("Expected error")
	}

	result := <-results
	if (len(result.Results)) != 2 {
		t.Fatal("Expected 2 check results, got: " + string(len(result.Results)))
	}
}

func TestCheckTasksWithPass(t *testing.T) {
	results := make(chan CheckResults, 1)
	defer close(results)
	task := checkProjectTask(mockCheckFactoryOnePass, "MockProject", "/", results)

	err := task()

	if err != nil {
		t.Fatal("Expected no error, got:", err)
	}

	result := <-results
	if (len(result.Results)) != 1 {
		t.Fatal("Expected 1 check results, got: " + string(len(result.Results)))
	}
}
