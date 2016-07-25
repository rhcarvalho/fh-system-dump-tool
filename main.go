// The RHMAP System Dump Tool.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// dumpDir is a path to the base directory where the output of the tool
	// goes.
	dumpDir = "rhmap-dumps"
	// dumpTimestampFormat is a layout for use with Time.Format. Used to
	// create directories with a timestamp. Based on time.RFC3339.
	dumpTimestampFormat = "2006-01-02T15-04-05Z0700"
)

// A Task performs some part of the RHMAP System Dump Tool.
type Task func() error

// An errorList accumulates multiple error messages and implements error.
type errorList []string

func (e errorList) Error() string {
	return "multiple errors:\n" + strings.Join(e, "\n")
}

// A projectResourceWriterFactory generates io.Writers for dumping data of a
// particular resource type within a project.
type projectResourceWriterFactory func(project, resource string) (io.Writer, error)

// A getProjectResourceCmdFactory generates commands to get resources of a given
// type in a project.
type getProjectResourceCmdFactory func(project, resource string) *exec.Cmd

// outToFile returns a function that creates an io.Writer that writes to a file
// in basepath with extension, given a project and resource.
func outToFile(basepath, extension string) projectResourceWriterFactory {
	return func(project, resource string) (io.Writer, error) {
		projectpath := filepath.Join(basepath, "projects", project)
		err := os.MkdirAll(projectpath, 0770)
		if err != nil {
			return nil, err
		}
		f, err := os.Create(filepath.Join(projectpath, resource+"."+extension))
		if err != nil {
			return nil, err
		}
		return f, nil
	}
}

// ResourceDefinitions fetches the JSON resource definition for all given types
// in project. For each resource type, it uses outFor and errOutFor to get
// io.Writers to write, respectively, the JSON output and any eventual error
// message.
func ResourceDefinitions(project string, types []string, outFor, errOutFor projectResourceWriterFactory) Task {
	return resourceDefinitions(func(project, resource string) *exec.Cmd {
		return exec.Command("oc", "-n", project, "get", resource, "-o=json")
	}, project, types, outFor, errOutFor)
}

func resourceDefinitions(cmdFactory getProjectResourceCmdFactory, project string, types []string, outFor, errOutFor projectResourceWriterFactory) Task {
	return func() error {
		var errors errorList
		// NOTE: we could fetch all resources of all types in a single
		// call to oc, by passing a comma-separated list of resource
		// types. Instead, we call oc multiple times to send the output
		// to different files without processing the contents of the
		// output from oc.
		for _, resource := range types {
			var err error
			cmd := cmdFactory(project, resource)
			cmd.Stdout, err = outFor(project, resource)
			if err != nil {
				errors = append(errors, err.Error())
				// Since we couldn't get an io.Writer for
				// cmd.Stdout, give up processing this resource
				// type, and skip to the next type.
				continue
			}
			var buf bytes.Buffer
			cmd.Stderr, err = errOutFor(project, resource)
			if err != nil {
				errors = append(errors, err.Error())
				// We can possibly try to run the command
				// without an io.Writer from errOutFor. In this
				// case, we'll attach an in-memory buffer so
				// that we can include the stderr output in
				// errors.
				cmd.Stderr = &buf
			} else {
				// Send stderr to both the io.Writer from
				// errOutFor, and an in-memory buffer, used to
				// enrich error messages.
				cmd.Stderr = io.MultiWriter(cmd.Stderr, &buf)
			}
			// TODO: limit the execution time with a timeout.
			err = cmd.Run()
			if err != nil {
				errors = append(errors, fmt.Sprintf("command %q: %v: %v", strings.Join(cmd.Args, " "), err, buf.String()))
				continue
			}
		}
		if len(errors) > 0 {
			return errors
		}
		return nil
	}
}

// GetProjects returns a list of project names visible by the current logged in
// user.
func GetProjects() ([]string, error) {
	return getProjects(exec.Command("oc", "get", "projects", "-o=jsonpath={.items[*].metadata.name}"))
}

// getProjects calls cmd, expected to output a space-separated list of project
// names to stdout, and returns a list of project names.
func getProjects(cmd *exec.Cmd) ([]string, error) {
	var projects []string
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("command %q: %v", strings.Join(cmd.Args, " "), err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		projects = append(projects, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("command %q: %v: %v", strings.Join(cmd.Args, " "), err, buf.String())
	}
	return projects, nil
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

func main() {
	start := time.Now().UTC()
	startTimestamp := start.Format(dumpTimestampFormat)
	basepath := filepath.Join(dumpDir, startTimestamp)

	var tasks []Task

	var resources = []string{"deploymentconfigs", "pods", "services", "events"}

	// Add tasks to fetch resource definitions.
	projects, err := GetProjects()
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	for _, p := range projects {
		outFor := outToFile(basepath, "json")
		errOutFor := outToFile(basepath, "stderr")
		task := ResourceDefinitions(p, resources, outFor, errOutFor)
		tasks = append(tasks, task)
	}

	fmt.Println("Starting RHMAP System Dump Tool...")
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for _, task := range tasks {
		task := task
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			task()
			fmt.Print(".")
			<-sem
		}()
	}
	wg.Wait()
	fmt.Printf("\nDumped system information to: %s\n", dumpDir)
}
