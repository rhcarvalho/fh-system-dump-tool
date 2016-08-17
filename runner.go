package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// A Runner runs commands.
type Runner interface {
	Run(cmd *exec.Cmd, path string) error
}

// DumpRunner is a Runner that dumps command execution output to disk.
type DumpRunner struct {
	dir string
}

var _ Runner = (*DumpRunner)(nil)

// NewDumpRunner creates a DumpRunner.
func NewDumpRunner(dir string) *DumpRunner {
	return &DumpRunner{
		dir: dir,
	}
}

// Run runs cmd and saves the stdout to path, relative to r.dir. Stderr, if any,
// goes to path.stderr. Parent directories are created if necessary. Path cannot
// be empty.
func (r *DumpRunner) Run(cmd *exec.Cmd, path string) error {
	if path == "" {
		return fmt.Errorf("command %q: missing path to output", strings.Join(cmd.Args, " "))
	}

	basedir := filepath.Join(r.dir, filepath.Dir(path))
	if err := os.MkdirAll(basedir, 0770); err != nil {
		return err
	}

	stdout, err := os.Create(filepath.Join(r.dir, path))
	if err != nil {
		return err
	}
	defer stdout.Close()

	var pathStderr = path + ".stderr"
	stderr := &lazyFileWriter{path: filepath.Join(r.dir, pathStderr)}
	defer stderr.Close()

	cmd.Stdout = io.MultiWriter(filterWriters(cmd.Stdout, stdout)...)
	cmd.Stderr = io.MultiWriter(filterWriters(cmd.Stderr, stderr)...)

	if err := cmd.Run(); err != nil {
		var b []byte
		if stderr.file != nil {
			stderr.file.Seek(0, os.SEEK_SET)
			b, _ = ioutil.ReadAll(stderr.file)
		}
		return fmt.Errorf("command %q: %s: %s", strings.Join(cmd.Args, " "), err, b)
	}
	return nil
}

// filterWriters filters out nil writers.
func filterWriters(writers ...io.Writer) []io.Writer {
	ws := make([]io.Writer, len(writers))
	copy(ws, writers)
	filtered := ws[:0]
	for _, w := range ws {
		if w != nil {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

// lazyFileWriter is an io.WriteCloser that writes to a file, but defers file
// creation up until the first write.
type lazyFileWriter struct {
	path string

	once sync.Once // ensures we attempt to create file only once.
	file *os.File
}

var _ io.WriteCloser = (*lazyFileWriter)(nil)

func (w *lazyFileWriter) Write(p []byte) (n int, err error) {
	w.once.Do(func() {
		w.file, err = os.Create(w.path)
	})
	if err != nil {
		return
	}
	return w.file.Write(p)
}

func (w *lazyFileWriter) Close() error {
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}
