package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// A Task performs some part of the RHMAP System Dump Tool.
type Task func() error

// RunAllDumpTasks runs all tasks known to the dump tool using concurrent workers.
// Dump output goes to path.
func RunAllDumpTasks(runner Runner, path string, workers int) {
	start := time.Now()

	tasks := GetAllDumpTasks(runner, path)
	results := make(chan error)

	// Start worker goroutines to run tasks concurrently.
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for task := range tasks {
				results <- task()
			}
		}()
	}
	// Wait for all workers to terminate, then close the results channel to
	// communicate that no more results will be sent.
	go func() {
		wg.Wait()
		close(results)
	}()

	taskCount := 0

	// Loop through the task execution results and log errors.
	for err := range results {
		taskCount++
		if err != nil {
			// TODO: there should be a way to identify which task
			// had an error.
			fmt.Fprintln(os.Stderr)
			log.Printf("Task error: %v", err)
			continue
		}
		fmt.Fprint(os.Stderr, ".")
	}
	fmt.Fprintln(os.Stderr)

	delta := time.Since(start)
	// Remove sub-second precision.
	delta -= delta % time.Second
	log.Printf("Run %d dump tasks in %v.", taskCount, delta)
}

// GetAllDumpTasks returns a channel of all tasks known to the dump tool. It returns
// immediately and sends tasks to the channel in a separate goroutine. The
// channel is closed after all tasks are sent.
// FIXME: GetAllDumpTasks should not need to know about basepath.
func GetAllDumpTasks(runner Runner, basepath string) <-chan Task {
	tasks := make(chan Task)
	go func() {
		defer close(tasks)

		projects, err := GetProjects(runner)
		if err != nil {
			tasks <- NewError(err)
			return
		}
		if len(projects) == 0 {
			tasks <- NewError(errors.New("no projects visible to the currently logged in user"))
			return
		}

		var wg sync.WaitGroup

		// Add tasks to fetch OpenShift metadata.
		wg.Add(1)
		go func() {
			defer wg.Done()
			GetOpenShiftMetadataTasks(tasks, runner, projects)
		}()

		// Add tasks to fetch resource definitions.
		wg.Add(1)
		go func() {
			defer wg.Done()

			resources := []string{
				"deploymentconfigs", "pods", "services",
				"events", "persistentvolumeclaims",
			}
			GetResourceDefinitionTasks(tasks, runner, projects, resources)

			// For cluster-scoped resources we need only one task to
			// fetch all definitions, instead of one per project.
			clusterScoped := []string{"persistentvolumes", "nodes"}
			for _, resource := range clusterScoped {
				tasks <- ResourceDefinition(runner, "", resource)
			}
		}()

		// Add tasks to fetch logs.
		wg.Add(1)
		go func() {
			defer wg.Done()
			// We should only care about logs for pods, because they
			// cover all other possible types.
			resourcesWithLogs := []string{"pods"}
			// FIXME: we should not be accessing a flag value
			// (global) here, instead take maxLines as an argument.
			GetFetchLogsTasks(tasks, runner, projects, resourcesWithLogs, *maxLogLines)
		}()

		// Add tasks to fetch Nagios data.
		wg.Add(1)
		go func() {
			defer wg.Done()
			GetNagiosTasks(tasks, runner, projects)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			GetMillicoreConfigTasks(tasks, runner, projects, getResourceNamesBySubstr)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			tasks <- GetOcAdmDiagnosticsTask(runner)
		}()
		wg.Wait()
	}()
	return tasks
}

func RunAllAnalysisTasks(runner Runner, path string, workers int) {
	start := time.Now()

	checkResults := make(chan CheckResults)
	tasks := GetAllAnalysisTasks(runner, path, checkResults)
	results := make(chan error)

	// Start worker goroutines to run tasks concurrently.
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for task := range tasks {
				results <- task()
			}
		}()
	}

	// Listen to the checkResults channel and write all the results into
	// the analysis.json file
	go func() {
		analysisResults := map[string][]Result{}
		filepath := filepath.Join(path, "analysis.json")
		err := os.MkdirAll(path, 0770)
		if err != nil {
			results <- err
			return
		}

		for result := range checkResults {
			analysisResults[result.Scope] = result.Results
			output, err := json.MarshalIndent(analysisResults, "", "    ")
			if err != nil {
				results <- err
			}
			ioutil.WriteFile(filepath, []byte(output), 0644)
		}
	}()

	// Wait for all workers to terminate, then close the results channel to
	// communicate that no more results will be sent.
	go func() {
		wg.Wait()
		close(checkResults)
		close(results)
	}()

	taskCount := 0

	// Loop through the task execution results and log errors.
	for err := range results {
		taskCount++
		if err != nil {
			// TODO: there should be a way to identify which task
			// had an error.
			fmt.Fprintln(os.Stderr)
			log.Printf("Task error: %v", err)
			continue
		}
		fmt.Fprint(os.Stderr, ".")
	}
	fmt.Fprintln(os.Stderr)

	delta := time.Since(start)
	// Remove sub-second precision.
	delta -= delta % time.Second
	log.Printf("Run %d analysis tasks in %v.", taskCount, delta)
}

func GetAllAnalysisTasks(runner Runner, path string, results chan<- CheckResults) <-chan Task {
	tasks := make(chan Task)
	go func() {
		defer close(tasks)

		projects, err := GetProjects(runner)
		if err != nil {
			tasks <- NewError(err)
			return
		}

		GetAnalysisTasks(tasks, path, projects, results)

	}()

	return tasks
}

// NewError returns a Task that always return the given error.
func NewError(err error) Task {
	return func() error { return err }
}
