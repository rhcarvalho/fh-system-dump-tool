package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"testing"
)

var b *bytes.Buffer

func mockOutFor(project, resource string) (io.Writer, io.Closer, error) {
	return b, ioutil.NopCloser(nil), nil
}

func mockCheckFactoryOnePassOneFail() []CheckTask {
	return []CheckTask{mockTestOne, mockTestTwo}
}
func mockCheckFactoryOnePass() []CheckTask {
	return []CheckTask{mockTestOne}
}

func mockTestOne(project string, stdErr io.Writer) (Result, error) {
	result := Result{StatusMessage: "Called mockTestOne"}
	return result, nil

}

func mockTestTwo(project string, stdErr io.Writer) (Result, error) {
	result := Result{StatusMessage: "Called mockTestTwo"}
	return result, errors.New("FAIL")
}

func TestCheckTasks(t *testing.T) {
	b = bytes.NewBuffer([]byte{})
	task := checkTasks(mockCheckFactoryOnePassOneFail, "MockProject", mockOutFor, mockOutFor)
	err := task()

	if err == nil {
		t.Fatal("Expected error")
	}

	if len(string(b.Bytes())) == 0 {
		t.Fatal("Tests not called")

	}

	b = bytes.NewBuffer([]byte{})
	task = checkTasks(mockCheckFactoryOnePass, "MockProject", mockOutFor, mockOutFor)
	err = task()

	if err != nil {
		t.Fatal("Expected no errors")
	}

	if len(string(b.Bytes())) == 0 {
		t.Fatal("Tests not called")

	}

}
