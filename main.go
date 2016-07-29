// The RHMAP System Dump Tool.
package main

import (
	"bufio"
	"bytes"
	"flag"
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

var maxParallelTasks = flag.Int("p", runtime.NumCPU(), "max number of tasks to run in parallel")

// A Task performs some part of the RHMAP System Dump Tool.
type Task func() error

// An errorList accumulates multiple error messages and implements error.
type errorList []string

func (e errorList) Error() string {
	return "multiple errors:\n" + strings.Join(e, "\n")
}

// A projectResourceWriterFactory generates io.Writers for dumping data of a
// particular resource type within a project.
type projectResourceWriterCloserFactory func(project, resource string) (io.Writer, io.Closer, error)

// outToFile returns a function that creates an io.Writer that writes to a file
// in basepath with extension, given a project and resource.
func outToFile(basepath, extension string) projectResourceWriterCloserFactory {
	return func(project, resource string) (io.Writer, io.Closer, error) {
		projectpath := filepath.Join(basepath, "projects", project)
		err := os.MkdirAll(projectpath, 0770)
		if err != nil {
			return nil, nil, err
		}
		f, err := os.Create(filepath.Join(projectpath, resource+"."+extension))
		if err != nil {
			return nil, nil, err
		}
		return f, f, nil
	}
}

// OutToTGZ returns an anonymous factory function that will create an io.Writer which writes into the tar archive
// provided. The path inside the tar.gz file is calculated from the project and resource provided
func outToTGZ(extension string, tarFile *Archive) projectResourceWriterCloserFactory {
	return func(project, resource string) (io.Writer, io.Closer, error) {
		projectPath := filepath.Join("projects", project)
		writer := tarFile.GetWriterToFile(filepath.Join(projectPath, resource+"."+extension))
		return writer, writer, nil
	}
}

func runCmdCaptureOutput(cmd *exec.Cmd, project, resource string, outFor, errOutFor projectResourceWriterCloserFactory) error {
	var err error
	var stdoutCloser, stderrCloser io.Closer

	cmd.Stdout, stdoutCloser, err = outFor(project, resource)
	if err != nil {
		// Since we couldn't get an io.Writer for cmd.Stdout, give up
		// processing this resource type.
		return err
	}
	defer stdoutCloser.Close()

	var buf bytes.Buffer
	cmd.Stderr, stderrCloser, err = errOutFor(project, resource)
	if err != nil {
		// We can possibly try to run the command without an io.Writer
		// from errOutFor. In this case, we'll attach an in-memory
		// buffer so that we can include the stderr output in errors.
		cmd.Stderr = &buf
	} else {
		defer stderrCloser.Close()
		// Send stderr to both the io.Writer from errOutFor, and an
		// in-memory buffer, used to enrich error messages.
		cmd.Stderr = io.MultiWriter(cmd.Stderr, &buf)
	}

	// TODO: limit the execution time with a timeout.
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("command %q: %v: %v", strings.Join(cmd.Args, " "), err, buf.String())
	}
	return nil
}

// GetProjects returns a list of project names visible by the current logged in
// user.
func GetProjects() ([]string, error) {
	return getSpaceSeparated(exec.Command("oc", "get", "projects", "-o=jsonpath={.items[*].metadata.name}"))
}

// getSpaceSeparated calls cmd, expected to output a space-separated list of
// words to stdout, and returns the words.
func getSpaceSeparated(cmd *exec.Cmd) ([]string, error) {
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
	flag.Parse()
	if !(*maxParallelTasks > 0) {
		printError(fmt.Errorf("argument to -p flag must be greater than 0"))
		os.Exit(1)
	}

	start := time.Now().UTC()
	startTimestamp := start.Format(dumpTimestampFormat)
	basepath := filepath.Join(dumpDir, startTimestamp)

	archiveFile, err := os.Create(basepath + ".tar.gz")
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	defer archiveFile.Close()

	tarFile, err := NewTgz(archiveFile)
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	defer tarFile.Close()

	var tasks []Task

	var resources = []string{"deploymentconfigs", "pods", "services", "events"}

	projects, err := GetProjects()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Add tasks to fetch resource definitions.
	for _, p := range projects {
		outFor := outToTGZ("json", tarFile)
		errOutFor := outToTGZ("stderr", tarFile)
		task := ResourceDefinitions(p, resources, outFor, errOutFor)
		tasks = append(tasks, task)
	}

	fmt.Println("Starting RHMAP System Dump Tool...")
	defer fmt.Printf("\nDumped system information to: %s\n", dumpDir)

	// Avoid the creating goroutines and other controls if we're executing
	// tasks sequentially.
	if *maxParallelTasks == 1 {
		for _, task := range tasks {
			task()
			fmt.Print(".")
		}
		return
	}
	// Run at most N tasks in parallel, and wait for all of them to
	// complete.
	var wg sync.WaitGroup
	sem := make(chan struct{}, *maxParallelTasks)
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
}
