package main

// A Task performs some part of the RHMAP System Dump Tool.
type Task func() error

// GetAllTasks returns a list of all tasks performed by the dump tool. It may
// return tasks even in the presence of an error.
// FIXME: GetAllTasks should not know about tarFile.
func GetAllTasks(tarFile *Archive) ([]Task, error) {
	var (
		tasks  []Task
		errors errorList
	)

	var (
		resources         = []string{"deploymentconfigs", "pods", "services", "events"}
		resourcesWithLogs = []string{"deploymentconfigs", "pods"}
	)

	projects, err := GetProjects()
	if err != nil {
		return nil, err
	}

	// Add tasks to fetch resource definitions.
	definitionsTasks, err := GetResourceDefinitionsTasks(projects, resources, tarFile)
	if err != nil {
		errors = append(errors, err)
	}
	tasks = append(tasks, definitionsTasks...)

	// Add tasks to fetch logs.
	logsTasks, err := GetFetchLogsTasks(projects, resourcesWithLogs, tarFile)
	if err != nil {
		errors = append(errors, err)
	}
	tasks = append(tasks, logsTasks...)

	if len(errors) > 0 {
		return tasks, errors
	}
	return tasks, nil
}

// GetResourceDefinitionsTasks returns a list of tasks to fetch the definitions
// of all resources in all projects.
// FIXME: GetResourceDefinitionsTasks should not know about tarFile.
func GetResourceDefinitionsTasks(projects, resources []string, tarFile *Archive) ([]Task, error) {
	var tasks []Task
	for _, p := range projects {
		outFor := outToTGZ("definitions", "json", tarFile)
		errOutFor := outToTGZ("definitions", "stderr", tarFile)
		task := ResourceDefinitions(p, resources, outFor, errOutFor)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetFetchLogsTasks returns a list of tasks to fetch resource logs. It may
// return tasks even in the presence of an error.
// FIXME: GetFetchLogsTasks should not know about tarFile.
func GetFetchLogsTasks(projects, resources []string, tarFile *Archive) ([]Task, error) {
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
			out, outCloser, _ := outToTGZ("logs", "logs", tarFile)(r.Project, name)
			errOut, errOutCloser, _ := outToTGZ("logs", "stderr", tarFile)(r.Project, name)
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
			out, outCloser, _ := outToTGZ("logs-previous", "logs", tarFile)(r.Project, name)
			errOut, errOutCloser, _ := outToTGZ("logs-previous", "stderr", tarFile)(r.Project, name)
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
