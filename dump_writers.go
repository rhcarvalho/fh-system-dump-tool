package main

import (
	"io"
	"os"
	"path/filepath"
)

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
