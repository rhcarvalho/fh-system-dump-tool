package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func mockAnalysisNoErrors(p string, d interface{}) error {
	contents := `{
		"projects": [{
			"project": "dev",
			"checks": [{
				"checkName": "check event log for errors",
				"ok": true,
				"message": "this issue was not detected"
			}, {
				"checkName": "check number of replicas in deployment configs",
				"ok": true,
				"message": "this issue was not detected"
			}, {
				"checkName": "check pods for containers in waiting state",
				"ok": true,
				"message": "this issue was not detected"
			}]
		}]
	}`

	decoder := json.NewDecoder(bytes.NewBuffer([]byte(contents)))
	err := decoder.Decode(&d)
	if err != nil {
		return err
	}
	return nil
}

func mockAnalysisErrors(p string, d interface{}) error {
	contents := `{
		"projects": [{
			"project": "rhmap-core",
			"checks": [{
				"checkName": "check event log for errors",
				"ok": false,
				"message": "Errors detected in event log",
				"events": [{
					"kind": "Event",
					"involvedObject": {
						"namespace": "rhmap-core",
						"name": "fh-ngui"
					},
					"reason": "FailedUpdate",
					"message": "Cannot update deployment rhmap-core/fh-ngui-3 status to Pending: replicationcontrollers \"fh-ngui-3\" cannot be updated: the object has been modified; please apply your changes to the latest version and try again",
					"count": 1,
					"type": "Warning"
				}]
			}]
		}]
	}`

	decoder := json.NewDecoder(bytes.NewBuffer([]byte(contents)))
	err := decoder.Decode(&d)
	if err != nil {
		return err
	}
	return nil
}

func TestRunOutputTaskNoErrors(t *testing.T) {
	o := bytes.NewBuffer([]byte{})
	e := bytes.NewBuffer([]byte{})
	var analysisResult AnalysisResult
	if err := mockAnalysisNoErrors("", &analysisResult); err != nil {
		t.Fatal(err)
	}
	RunOutputTask(o, e, analysisResult)

	if got := len(o.Bytes()); got > 0 {
		t.Fatal("len(o.Bytes()), expected: 0, got:", got)
	}
	if got := len(e.Bytes()); got > 0 {
		t.Fatal("len(e.Bytes()), expected: 0, got:", got)
	}
}

func TestRunOutputTaskFoundErrors(t *testing.T) {
	o := bytes.NewBuffer([]byte{})
	e := bytes.NewBuffer([]byte{})
	var analysisResult AnalysisResult
	if err := mockAnalysisErrors("", &analysisResult); err != nil {
		t.Fatal(err)
	}
	RunOutputTask(o, e, analysisResult)

	if !strings.Contains(string(o.Bytes()), "rhmap-core") {
		t.Fatal("string(o.Bytes()), expected: to contain 'rhmap-core', got:", string(o.Bytes()))
	}
	if got := len(e.Bytes()); got > 0 {
		t.Fatal("len(e.Bytes()), expected: 0, got:", got)
	}
}
