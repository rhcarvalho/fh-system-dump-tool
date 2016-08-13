// The RHMAP System Dump Tool.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
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
	// defaultMaxLogLines is the default limit of number of log lines to
	// fetch.
	defaultMaxLogLines = 1000

	// The version of the fh-system-dump-tool
	version = "0.1.0"
)

var (
	maxParallelTasks = flag.Int("p", runtime.NumCPU(), "max number of tasks to run in parallel")
	maxLogLines      = flag.Int("max-log-lines", defaultMaxLogLines, "max number of log lines fetched with oc logs")
	printVersion     = flag.Bool("version", false, "print version and exit")
)

func runCmdCaptureOutput(cmd *exec.Cmd, out, errOut io.Writer) error {
	cmd.Stdout = out

	// Send stderr to an in-memory buffer used to enrich error messages.
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if errOut != nil {
		// If errOut is non-nil, also send stderr to it.
		cmd.Stderr = io.MultiWriter(cmd.Stderr, errOut)
	}

	// TODO: limit the execution time with a timeout.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q: %v: %v", strings.Join(cmd.Args, " "), err, buf.String())
	}
	return nil
}

func runCmdCaptureOutputDeprecated(cmd *exec.Cmd, project, resource string, outFor, errOutFor projectResourceWriterCloserFactory) error {
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

// GetResourceNames returns a list of resource names of type rtype, visible by
// the current logged in user, scoped by project.
func GetResourceNames(project, rtype string) ([]string, error) {
	return getSpaceSeparated(exec.Command("oc", "-n", project, "get", rtype, "-o=jsonpath={.items[*].metadata.name}"))
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

	if *printVersion {
		fmt.Println("RHMAP fh-system-dump-tool v" + version)
		os.Exit(0)
	}

	if !(*maxParallelTasks > 0) {
		printError(fmt.Errorf("argument to -p flag must be greater than 0"))
		os.Exit(1)
	}

	log.Println("Starting RHMAP System Dump Tool...")

	start := time.Now().UTC()
	startTimestamp := start.Format(dumpTimestampFormat)

	basePath := filepath.Join(dumpDir, startTimestamp)

	log.Println("Preparing tasks...")

	tasks, err := GetAllTasks(basePath)
	if err != nil {
		printError(err)
		defer os.Exit(1)
	}
	if len(tasks) == 0 {
		log.Print("No tasks found to execute.")
		return
	}

	// defer creating a tar.gz file from the dumped output files
	defer func() {
		var stdout, stderr bytes.Buffer

		cmd := exec.Command("tar", "-czf", basePath+".tar.gz", basePath)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Dumped system information to: %s", basePath)
			return
		}

		// The archive was created successfully, remove basePath. The
		// error from os.RemoveAll is intentionally ignored, since there
		// is no useful action we can do, and we don't need to confuse
		// the user with an error message.
		os.RemoveAll(basePath)

		log.Printf("Dumped system information to: %s", basePath+".tar.gz")
	}()

	log.Println("Running tasks...")

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
			fmt.Fprint(os.Stderr, ".")
			<-sem
		}()
	}
	wg.Wait()

	fmt.Fprintln(os.Stderr)
}
