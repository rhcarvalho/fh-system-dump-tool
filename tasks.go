package main

import "errors"

// A Task performs some part of the RHMAP System Dump Tool.
type Task func() error

// GetAllTasks returns a list of all tasks performed by the dump tool. It may
// return tasks even in the presence of an error.
// FIXME: GetAllTasks should not need to know about basepath.
func GetAllTasks(basepath string) ([]Task, error) {
	var (
		tasks     []Task
		retErrors errorList
	)

	var (
		resources         = []string{"deploymentconfigs", "pods", "services", "events"}
		resourcesWithLogs = []string{"deploymentconfigs", "pods"}
	)

	projects, err := GetProjects()
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, errors.New("no projects visible to the currently logged in user")
	}

	// Add tasks to fetch resource definitions.
	definitionsTasks, err := GetResourceDefinitionsTasks(projects, resources, basepath)
	if err != nil {
		retErrors = append(retErrors, err)
	}
	tasks = append(tasks, definitionsTasks...)

	// Add tasks to fetch logs.
	logsTasks, err := GetFetchLogsTasks(projects, resourcesWithLogs, basepath)
	if err != nil {
		retErrors = append(retErrors, err)
	}
	tasks = append(tasks, logsTasks...)

	nagiosDataTasks, err := GetNagiosTasks(projects, basepath)
	if err != nil {
		retErrors = append(retErrors, err)
	}
	tasks = append(tasks, nagiosDataTasks...)

	// Add check tasks
	for _, p := range projects {
		outFor := outToFile(basepath, "json", "analysis")
		errOutFor := outToFile(basepath, "stderr", "analysis")
		task := CheckTasks(p, outFor, errOutFor)
		tasks = append(tasks, task)
	}

	if len(retErrors) > 0 {
		return tasks, retErrors
	}
	return tasks, nil
}

// GetNagiosTasks will return an array of Tasks each of which will dump the nagios data for one project.
func GetNagiosTasks(projects []string, basepath string) ([]Task, error) {
	var tasks []Task
	var errors errorList
	for _, p := range projects {
		pods, err := getResourceNamesBySubstr(p, "pod", "nagios")
		if err != nil {
			errors = append(errors, err)
			continue
		}
		for _, pod := range pods {
			outFor := outToFile(basepath, "dat", "nagios")
			errOutFor := outToFile(basepath, "stderr", "nagios")
			task := GetNagiosStatusData(p, pod, outFor, errOutFor)
			tasks = append(tasks, task)

			outFor = outToFile(basepath, "tar", "nagios")
			errOutFor = outToFile(basepath, "stderr", "nagios")
			task = GetNagiosHistoricalData(p, pod, outFor, errOutFor)
			tasks = append(tasks, task)
		}
	}
	if len(errors) > 0 {
		return tasks, errors
	}
	return tasks, nil
}

// GetResourceDefinitionsTasks returns a list of tasks to fetch the definitions
// of all resources in all projects.
// FIXME: GetResourceDefinitionsTasks should not know about basepath.
func GetResourceDefinitionsTasks(projects, resources []string, basepath string) ([]Task, error) {
	var tasks []Task
	for _, p := range projects {
		outFor := outToFile(basepath, "json", "definitions")
		errOutFor := outToFile(basepath, "stderr", "definitions")
		task := ResourceDefinitions(p, resources, outFor, errOutFor)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetFetchLogsTasks returns a list of tasks to fetch resource logs. It may
// return tasks even in the presence of an error.
// FIXME: GetFetchLogsTasks should not need to know about the output directory.
func GetFetchLogsTasks(projects, resources []string, basepath string) ([]Task, error) {
	var (
		tasks  []Task
		errors errorList
	)
	loggableResources, err := GetLogabbleResources(projects, resources)
	if err != nil {
		errors = append(errors, err)
	}
	if len(loggableResources) == 0 {
		return nil, errors
	}
	for _, r := range loggableResources {
		r := r
		name := r.Type + "-" + r.Name
		if r.Container != "" {
			name += "-" + r.Container
		}
		// Add tasks to fetch current logs.
		{
			// FIXME: Do not ignore errors.
			out, outCloser, _ := outToFile(basepath, "logs", "logs")(r.Project, name)
			errOut, errOutCloser, _ := outToFile(basepath, "stderr", "logs")(r.Project, name)
			task := func() error {
				defer outCloser.Close()
				defer errOutCloser.Close()
				return FetchLogs(r, *maxLogLines, out, errOut)()
			}
			tasks = append(tasks, task)
		}
		// Add tasks to fetch previous logs.
		{
			// FIXME: Do not ignore errors.
			out, outCloser, _ := outToFile(basepath, "logs", "logs-previous")(r.Project, name)
			errOut, errOutCloser, _ := outToFile(basepath, "stderr", "logs-previous")(r.Project, name)
			task := func() error {
				defer outCloser.Close()
				defer errOutCloser.Close()
				return FetchPreviousLogs(r, *maxLogLines, out, errOut)()
			}
			tasks = append(tasks, task)
		}
	}
	if len(errors) > 0 {
		return tasks, errors
	}
	return tasks, nil
}

// GetLogabbleResources returns a list of loggable resources. It may return
// results even in the presence of an error.
func GetLogabbleResources(projects, resources []string) ([]LoggableResource, error) {
	var (
		loggableResources []LoggableResource
		errors            errorList
	)
	for _, p := range projects {
		for _, rtype := range resources {
			names, err := GetResourceNames(p, rtype)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			for _, name := range names {
				resources, err := GetLoggableResources(p, rtype, name)
				if err != nil {
					errors = append(errors, err)
					continue
				}
				loggableResources = append(loggableResources, resources...)
			}
		}
	}
	if len(errors) > 0 {
		return loggableResources, errors
	}
	return loggableResources, nil
}
