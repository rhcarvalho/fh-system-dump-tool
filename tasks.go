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

	var resources = []string{"deploymentconfigs", "pods", "services", "events"}

	projects, err := GetProjects()
	if err != nil {
		return nil, err
	}

	// Add tasks to fetch resource definitions.
	definitionsTasks, err := GetResourceDefinitionsTasks(projects, resources, tarFile)
	if err != nil {
		errors = append(errors, err.Error())
	}
	tasks = append(tasks, definitionsTasks...)

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
		outFor := outToTGZ("json", tarFile)
		errOutFor := outToTGZ("stderr", tarFile)
		task := ResourceDefinitions(p, resources, outFor, errOutFor)
		tasks = append(tasks, task)
	}
	return tasks, nil
}
