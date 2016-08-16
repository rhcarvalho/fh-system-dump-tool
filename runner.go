package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// A Runner runs commands.
type Runner interface {
	Run(cmd *exec.Cmd, path string) error
}

// DumpRunner is a Runner that dumps command execution output and metadata to
// disk.
type DumpRunner struct {
	dir string

	once    sync.Once // used to create workDir if needed.
	workDir string

	// TODO: store every run, dump it to metadata.json in the end.
	mu   sync.Mutex
	runs []RunInfo
}

// RunInfo stores metadata from a command execution.
type RunInfo struct {
	Args []string

	StdoutPath string
	StderrPath string

	StartTime  time.Time
	EndTime    time.Time
	SystemTime time.Duration
	UserTime   time.Duration
	MaxRSS     int64
	ExitCode   int
}

var _ Runner = (*DumpRunner)(nil)

// NewDumpRunner creates a DumpRunner.
func NewDumpRunner(dir, workDir string) *DumpRunner {
	return &DumpRunner{
		dir:     dir,
		workDir: workDir,
	}
}

// Run runs cmd and saves the stdout to path, relative to r.dir. Stderr, if any,
// goes to path.stderr. Parent directories are created if necessary. Path cannot
// be empty. Each run also saves metadata to r.workDir.
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

	startTime := time.Now()
	defer func() {
		status := cmd.ProcessState.Sys().(syscall.WaitStatus)
		usage := cmd.ProcessState.SysUsage().(*syscall.Rusage)
		ri := RunInfo{
			Args:       cmd.Args,
			StdoutPath: path,
			StderrPath: pathStderr,
			StartTime:  startTime,
			EndTime:    time.Now(),
			SystemTime: cmd.ProcessState.SystemTime(),
			UserTime:   cmd.ProcessState.UserTime(),
			MaxRSS:     usage.Maxrss,
			ExitCode:   status.ExitStatus(),
		}
		b, err := json.MarshalIndent(ri, "", "  ")
		if err != nil {
			log.Printf("failed to serialize command execution metadata: %s: %s", strings.Join(cmd.Args, " "), err)
			return
		}
		name := filepath.Join(r.workDir, strconv.Itoa(cmd.Process.Pid))
		r.once.Do(func() {
			os.MkdirAll(filepath.Dir(name), 0770)
		})
		if err = ioutil.WriteFile(name, b, 0660); err != nil {
			log.Printf("failed to write command execution metadata: %s: %s", strings.Join(cmd.Args, " "), err)
			return
		}
	}()
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
