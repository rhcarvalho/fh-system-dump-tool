package main
import (
	"bytes"
	"io/ioutil"
	"io"
	"errors"
	"encoding/json"
	"strings"
)

type Info struct {
	PodName   string
	Namespace string
	Resource  string
	Count     int
	Entry     string
}

type Result struct {
	CheckName     string `json:"checkName" yaml:"checkName"`
	Status        int    `json:"status" yaml:"status"`
	StatusMessage string `json:"statusMessage" yaml:"statusMessage"`
	Info          []Info `json:"info" yaml:"info"`
}

type Events struct {
	Items []struct {
		InvolvedObject struct {
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		} `json:"involvedObject"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
		Count   int    `json:"count"`
	} `json:"items"`
}

type CheckTask func(string, projectResourceWriterCloserFactory, projectResourceWriterCloserFactory) error

// ResourceDefinitions is a task factory for tasks that fetch the JSON resource
// definition for all given types in project. For each resource type, the task
// uses outFor and errOutFor to get io.Writers to write, respectively, the JSON
// output and any eventual error message.
func CheckTasks(project string, checks []string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return checkTasks(func(project, check string) CheckTask {
		switch check{
		case "CheckImagePullBackOff":
			return CheckImagePullBackOff
		}
		return nil
	}, project, checks, outFor, errOutFor)
}

// A getProjectCheckFactory generates commands to get resources of a given
// type in a project.
type getProjectCheckFactory func(project, check string) CheckTask


func checkTasks(checkFactory getProjectCheckFactory, project string, checks []string, outFor, errOutFor projectResourceWriterCloserFactory) Task {
	return func() error {
		var errors errorList
		for _, checkName := range checks {
			check := checkFactory(project, checkName)
			err := check(project, outFor, errOutFor)
			if err != nil {
				errors = append(errors, err.Error())
			}

		}
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

func CheckImagePullBackOff(project string, outFor, errOutFor projectResourceWriterCloserFactory) error {
	stdOut, stdOutCloser, err := outFor(project, "checkImagePullBackOff")
	if err != nil {
		return err
	}
	defer stdOutCloser.Close()
	stdErr, stdErrCloser, err := errOutFor(project, "checkImagePullBackOff")
	if err != nil {
		return err
	}
	defer stdErrCloser.Close()

	eventsJson, err := getResourceReader(project, "events")
	if err != nil {
		stdErr.Write([]byte(err.Error()))
		return err
	}
	result := Result{}
	events := Events{}

	decoder := json.NewDecoder(eventsJson)
	err = decoder.Decode(&events)
	if err != nil {
		stdErr.Write([]byte(err.Error()))
		return err
	}

	for _, event := range events.Items {
		if event.Reason == "FailedSync" && strings.Contains(event.Message, "ImagePullBackOff") {
			info := Info{PodName: event.InvolvedObject.Name, Namespace: event.InvolvedObject.Namespace, Count: event.Count, Entry: event.Message, Resource: "events"}
			result.Status = 1
			result.StatusMessage = "This issue may be present in the system"
			result.Info = append(result.Info, info)
		}
	}

	output, err := json.MarshalIndent(result, "", "    ")

	stdOut.Write(output)

	return nil
}
