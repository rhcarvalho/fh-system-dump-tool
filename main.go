// The RHMAP System Dump Tool.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
)

var (
	concurrentTasks = flag.Int("p", runtime.NumCPU(), "number of tasks to run concurrently")
	maxLogLines     = flag.Int("max-log-lines", defaultMaxLogLines, "max number of log lines fetched with oc logs")
	printVersion    = flag.Bool("version", false, "print version and exit")
)

// GetProjects returns a list of project names visible by the current logged in
// user.
func GetProjects(runner Runner) ([]string, error) {
	cmd := exec.Command("oc", "get", "projects", "-o=jsonpath={.items[*].metadata.name}")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := runner.Run(cmd, filepath.Join("project-names")); err != nil {
		return nil, err
	}
	return readSpaceSeparated(&b)
}

// GetResourceNames returns a list of resource names of type rtype, visible by
// the current logged in user, scoped by project.
func GetResourceNames(runner Runner, project, rtype string) ([]string, error) {
	cmd := exec.Command("oc", "-n", project, "get", rtype, "-o=jsonpath={.items[*].metadata.name}")
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := runner.Run(cmd, filepath.Join("projects", project, "names", rtype)); err != nil {
		return nil, err
	}
	return readSpaceSeparated(&b)
}

// readSpaceSeparated reads from r and returns a list of space-separated words.
func readSpaceSeparated(r io.Reader) ([]string, error) {
	var words []string
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return words, nil
}

func main() {
	flag.Parse()

	if *printVersion {
		PrintVersion(os.Stdout)
		os.Exit(0)
	}

	if !(*concurrentTasks > 0) {
		fmt.Fprintln(os.Stderr, "Error: argument to -p flag must be greater than 0")
		os.Exit(1)
	}

	start := time.Now().UTC()
	startTimestamp := start.Format(dumpTimestampFormat)

	basePath := filepath.Join(dumpDir, startTimestamp)

	if err := os.MkdirAll(basePath, 0770); err != nil {
		log.Fatalln("Error:", err)
	}

	var b bytes.Buffer
	PrintVersion(&b)
	if err := ioutil.WriteFile(filepath.Join(basePath, "version"), b.Bytes(), 0660); err != nil {
		log.Fatalln("Error:", err)
	}

	logfile, err := os.Create(filepath.Join(basePath, "dump.log"))
	if err != nil {
		log.Fatalln("Error:", err)
	}
	defer logfile.Close()
	log.SetOutput(io.MultiWriter(os.Stderr, logfile))
	fileOnlyLogger := log.New(logfile, "", log.LstdFlags)

	// defer creating a tar.gz file from the dumped output files
	defer func() {
		// Write this only to logfile, before we archive it and remove
		// basePath. After that, logs will go only to stderr.
		fileOnlyLogger.Printf("Dumped system information to: %s", basePath)

		var stdout, stderr bytes.Buffer

		cmd := exec.Command("tar", "-czf", basePath+".tar.gz", basePath)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			log.Printf("There was an error creating archiving the dumped data, so the dumped system information is stored unarchived in: %s", basePath)
			return
		}

		// The archive was created successfully, remove basePath. The
		// error from os.RemoveAll is intentionally ignored, since there
		// is no useful action we can do, and we don't need to confuse
		// the user with an error message.
		os.RemoveAll(basePath)

		log.Printf("Dumped system information to: %s", basePath+".tar.gz")
	}()

	log.Print("Starting RHMAP System Dump Tool...")

	runner := NewDumpRunner(basePath)

	log.Print("Running dump tasks...")
	RunAllDumpTasks(runner, basePath, *concurrentTasks)
	log.Print("Running analysis tasks...")
	RunAllAnalysisTasks(runner, basePath, *concurrentTasks)
}
