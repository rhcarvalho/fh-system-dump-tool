package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func mockCheckFactoryOnePassOneFail() []CheckTask {
	return []CheckTask{mockTestOne, mockTestTwo}
}
func mockCheckFactoryOnePass() []CheckTask {
	return []CheckTask{mockTestOne}
}

func mockJSONResourceFactory(p []string, d interface{}) error {
	return nil
}

func mockTestOne(loader DumpedJSONResourceFactory) (Result, error) {
	result := Result{StatusMessage: "Called mockTestOne"}
	return result, nil
}

func mockTestTwo(loader DumpedJSONResourceFactory) (Result, error) {
	result := Result{StatusMessage: "Called mockTestTwo"}
	return result, errors.New("FAIL")
}

func TestCheckTasksWithFail(t *testing.T) {
	results := make(chan CheckResults, 1)
	defer close(results)
	task := checkProjectTask(mockCheckFactoryOnePassOneFail, mockJSONResourceFactory, "MockProject", results)

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
	task := checkProjectTask(mockCheckFactoryOnePass, mockJSONResourceFactory, "MockProject", results)

	err := task()

	if err != nil {
		t.Fatal("Expected no error, got:", err)
	}

	result := <-results
	if (len(result.Results)) != 1 {
		t.Fatal("Expected 1 check results, got: " + string(len(result.Results)))
	}
}

func mockEventLogWithWarningFactory(p []string, d interface{}) error {
	contents := `{
		"kind": "List",
		"apiVersion": "v1",
		"metadata": {},
		"items": [
			{
				"kind": "Event",
				"apiVersion": "v1",
				"metadata": {
					"name": "mongodb-2-1-x66za.14691ab157eb4089",
					"namespace": "qe-3node-4-1",
					"selfLink": "/api/v1/namespaces/qe-3node-4-1/events/mongodb-2-1-x66za.14691ab157eb4089",
					"uid": "68dd0b95-5e16-11e6-a344-0abb8905d551",
					"resourceVersion": "9471053",
					"creationTimestamp": "2016-08-09T09:48:22Z",
					"deletionTimestamp": "2016-08-23T16:04:30Z"
				},
				"involvedObject": {
					"kind": "Pod",
					"namespace": "qe-3node-4-1",
					"name": "mongodb-2-1-x66za",
					"uid": "915b7b41-5d33-11e6-9064-0abb8905d551",
					"apiVersion": "v1",
					"resourceVersion": "7932636"
				},
				"reason": "FailedSync",
				"message": "Error syncing pod, skipping: API error (500): Unknown device 84d72d4cf06a1e292ba21c36c5c93638ccd3c4cfab8bf048bd434ff9cdf43722\n",
				"source": {
					"component": "kubelet",
					"host": "10.10.0.141"
				},
				"firstTimestamp": "2016-08-09T09:48:22Z",
				"lastTimestamp": "2016-08-23T14:04:30Z",
				"count": 94215,
				"type": "Warning"
			}
		]
	}`

	decoder := json.NewDecoder(bytes.NewBuffer([]byte(contents)))
	err := decoder.Decode(&d)
	if err != nil {
		return err
	}
	return nil
}

func MockDeployConfigWithReplicaZero(p []string, d interface{}) error {
	contents := `{
		"kind": "List",
		"apiVersion": "v1",
		"metadata": {},
		"items": [
			{
				"kind": "DeploymentConfig",
				"apiVersion": "v1",
				"metadata": {
					"name": "fh-mbaas",
					"namespace": "qe-3node-4-1",
					"labels": {
						"name": "fh-mbaas"
					}
				},
				"spec": {
					"replicas": 0,
					"selector": {
						"name": "fh-mbaas"
					},
					"template": {
						"metadata": {
							"labels": {
								"name": "fh-mbaas"
							}
						}
					}
				}
			}
		]
	}`

	decoder := json.NewDecoder(bytes.NewBuffer([]byte(contents)))
	err := decoder.Decode(&d)
	if err != nil {
		return err
	}
	return nil
}

func mockJSONResourceErrorFactory(p []string, d interface{}) error {
	return errors.New("mock error")
}

func TestCheckEventLogForErrors(t *testing.T) {
	res, err := CheckEventLogForErrors(mockEventLogWithWarningFactory)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 1 {
		t.Fatal("res.Status expected: 1, got:", res.Status)
	}
	if len(res.Events) != 1 {
		t.Fatal("len(res.Events) expected: 1, got:", len(res.Events))
	}
	if res.Events[0].Count != 94215 {
		t.Fatal("res.Events[0].Count expected: 94215, got:", string(res.Events[0].Count))
	}
	if res.Events[0].Type != "Warning" {
		t.Fatal("res.Events[0].Type expected: 'Warning', got: '" + res.Events[0].Type + "'")
	}
	if res.Events[0].Reason != "FailedSync" {
		t.Fatal("res.Events[0].Reason expected: 'FailedSync', got: '" + res.Events[0].Reason + "'")
	}

	res, err = CheckEventLogForErrors(mockJSONResourceErrorFactory)
	if err == nil {
		t.Fatal("CheckEventLogForErrors(mockJSONResourceErrorFactory) expected error, got none")
	}
	if res.Status != 2 {
		t.Fatal("res.Status expected 2, got:", res.Status)
	}
}

func TestCheckDeployConfigsReplicasNotZero(t *testing.T) {
	res, err := CheckDeployConfigsReplicasNotZero(MockDeployConfigWithReplicaZero)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 1 {
		t.Fatal("res.Status expected: 1, got:", res.Status)
	}
	if res.Info[0].Count != 1 {
		t.Fatal("res.Info[0].Count expected 1, got:" + string(res.Info[0].Count))
	}

	res, err = CheckDeployConfigsReplicasNotZero(mockJSONResourceErrorFactory)
	if err == nil {
		t.Fatal("CheckDeployConfigsReplicasNotZero(mockJSONResourceErrorFactory) expected error, got none")
	}
	if res.Status != 2 {
		t.Fatal("res.Status expected 2, got:", res.Status)
	}
}
