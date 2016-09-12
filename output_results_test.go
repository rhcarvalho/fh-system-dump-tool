package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func mockAnalysisNoErrors(p string, d interface{}) error {
	contents := `{
		"projects": {
			"dev": [
				{
					"checkName": "check eventlog for any errors",
					"status": 0,
					"statusMessage": "this issue was not detected",
					"info": [],
					"events": []
				},
				{
					"checkName": "check deployconfig replicas not 0",
					"status": 0,
					"statusMessage": "this issue was not detected",
					"info": [],
					"events": []
				},
				{
					"checkName": "check pods for 'waiting' containers",
					"status": 0,
					"statusMessage": "this issue was not detected",
					"info": [],
					"events": []
				}
			]
		}
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
		"projects": {
			"rhmap-core": [
				{
					"checkName": "check eventlog for any errors",
					"status": 1,
					"statusMessage": "Errors detected in event log",
					"info": [],
					"events": [
						{
							"kind": "Event",
							"involvedObject": {
								"namespace": "rhmap-core",
								"name": "fh-ngui"
							},
							"reason": "FailedUpdate",
							"message": "Cannot update deployment rhmap-core/fh-ngui-3 status to Pending: replicationcontrollers \"fh-ngui-3\" cannot be updated: the object has been modified; please apply your changes to the latest version and try again",
							"count": 1,
							"type": "Warning"
						}
					]
				}
			]
		}
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
	analysisResults := AnalysisResults{}
	if err := mockAnalysisNoErrors("", &analysisResults); err != nil {
		t.Fatal(err)
	}
	RunOutputTask(o, e, analysisResults)

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
	analysisResults := AnalysisResults{}
	if err := mockAnalysisErrors("", &analysisResults); err != nil {
		t.Fatal(err)
	}
	RunOutputTask(o, e, analysisResults)

	if !strings.Contains(string(o.Bytes()), "rhmap-core") {
		t.Fatal("string(o.Bytes()), expected: to contain 'rhmap-core', got:", string(o.Bytes()))
	}
	if got := len(e.Bytes()); got > 0 {
		t.Fatal("len(e.Bytes()), expected: 0, got:", got)
	}
}
