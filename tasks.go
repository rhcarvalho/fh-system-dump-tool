package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// A Task performs some part of the RHMAP System Dump Tool.
type Task func() error

// RunAllTasks runs all tasks known to the dump tool using concurrent workers.
// Dump output goes to path.
func RunAllTasks(runner Runner, path string, workers int) {
	start := time.Now()

	tasks := GetAllTasks(runner, path)
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
	log.Printf("Run %d tasks in %v.", taskCount, delta)
}

// GetAllTasks returns a channel of all tasks known to the dump tool. It returns
// immediately and sends tasks to the channel in a separate goroutine. The
// channel is closed after all tasks are sent.
// FIXME: GetAllTasks should not need to know about basepath.
func GetAllTasks(runner Runner, basepath string) <-chan Task {
	var (
		resources = []string{"deploymentconfigs", "pods", "services", "events"}
		// We should only care about logs for pods, because they cover
		// all other possible types.
		resourcesWithLogs = []string{"pods"}
	)
	tasks := make(chan Task)
	go func() {
		defer close(tasks)

		projects, err := GetProjects()
		if err != nil {
			tasks <- NewError(err)
			return
		}
		if len(projects) == 0 {
			tasks <- NewError(errors.New("no projects visible to the currently logged in user"))
			return
		}

		var wg sync.WaitGroup

		// Add tasks to fetch resource definitions.
		wg.Add(1)
		go func() {
			defer wg.Done()
			GetResourceDefinitionTasks(tasks, runner, projects, resources)
		}()

		// Add tasks to fetch logs.
		wg.Add(1)
		go func() {
			defer wg.Done()
			GetFetchLogsTasks(tasks, runner, projects, resourcesWithLogs)
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
			tasks <- GetOcAdmDiagnosticsTask(runner)
		}()

		wg.Wait()

		// After all other tasks are done, add analysis tasks. We want
		// to run them strictly later so that they can leverage the
		// output of commands executed previously by other tasks, e.g.,
		// reading resource definitions.
		for _, p := range projects {
			outFor := outToFile(basepath, "json", "analysis")
			errOutFor := outToFile(basepath, "stderr", "analysis")
			tasks <- CheckTasks(p, outFor, errOutFor)
		}
	}()
	return tasks
}

// NewError returns a Task that always return the given error.
func NewError(err error) Task {
	return func() error { return err }
}
