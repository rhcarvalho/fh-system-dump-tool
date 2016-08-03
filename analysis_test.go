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

func mockCheckFactory(project, check string) CheckTask {
	switch check {
	case "mockTestOne":
		return mockTestOne
	case "mockTestTwo":
		return mockTestTwo
	}
	return nil
}

func mockTestOne(project string, outFor, errOutFor projectResourceWriterCloserFactory) error {
	writer, closer, err := outFor(project, "mockResource")
	if err != nil {
		return err
	}
	writer.Write([]byte("Test One"))
	closer.Close()
	return nil

}

func mockTestTwo(project string, outFor, errOutFor projectResourceWriterCloserFactory) error {
	writer, closer, err := outFor(project, "mockResource")
	if err != nil {
		return err
	}
	writer.Write([]byte("Test Two"))
	closer.Close()
	return errors.New("FAIL")
}

func TestcheckTasks(t *testing.T) {
	b = bytes.NewBuffer([]byte{})
	task := checkTasks(mockCheckFactory, "MockProject", []string{"mockTestOne", "mockTestTwo"}, mockOutFor, mockOutFor)
	err := task()

	if err == nil {
		t.Fatal("Expected error")
	}

	if len(string(b.Bytes())) == 0 {
		t.Fatal("Tests not called")

	}

	b = bytes.NewBuffer([]byte{})
	task = checkTasks(mockCheckFactory, "MockProject", []string{"mockTestOne"}, mockOutFor, mockOutFor)
	err = task()

	if err != nil {
		t.Fatal("Expected no errors")
	}

	if len(string(b.Bytes())) == 0 {
		t.Fatal("Tests not called")

	}

}
