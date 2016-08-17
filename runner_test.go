package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Use `go test -tree` to debug DumpRunner tests.
var printTree = flag.Bool("tree", false, "print file tree in DumpRunner tests")

func TestDumpRunner(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-dumprunner-dir-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if *printTree {
		defer func() { t.Log("\n" + tree(dir)) }()
	}

	// We will use dr several times to make sure it does what it should do.
	// Each subtest begins with setting the cmd we will run.
	dr := NewDumpRunner(dir)

	// #1 -- common case, command exits with 0, output is saved to a file.
	cmd := helperCommand("echo", "ok")

	// Set cmd.Stdout to later ensure that dr.Run will write to both b and a
	// file.
	var b bytes.Buffer
	cmd.Stdout = &b

	// These are the files and contents we expect to see:
	names := []string{"out"}
	contents := []string{cmd.Args[len(cmd.Args)-1] + "\n"}

	if err := dr.Run(cmd, names[len(names)-1]); err != nil {
		t.Errorf("Run(%q) = %v, want %v", cmd.Args[3:], err, nil)
	}
	if err := dirHasExactFiles(dir, names, contents); err != nil {
		t.Error(err)
	}
	if got, want := b.String(), contents[len(contents)-1]; got != want {
		t.Errorf("original stdout = %q, want %q", got, want)
	}

	// #2 -- command exits with non-zero status, both stdout and stderr are
	// saved to disk.
	cmd = helperCommand("stderrfail")
	// We expect the command above to fail, and dr.Run should augment the
	// error message to include the stderr of the command.
	wantErrMsg := "exit status 1: some stderr text"

	// Set cmd.Stderr to later ensure that dr.Run will write to both b and a
	// file.
	b.Reset()
	cmd.Stderr = &b

	names = append(names, "out2", "out2.stderr")
	contents = append(contents, "", "some stderr text\n")

	if err := dr.Run(cmd, names[len(names)-2]); err == nil || !strings.Contains(err.Error(), wantErrMsg) {
		t.Errorf("Run(\"stderrfail\") = %v, want %v", err, wantErrMsg)
	}
	if err := dirHasExactFiles(dir, names, contents); err != nil {
		t.Error(err)
	}
	if got, want := b.String(), contents[len(contents)-1]; got != want {
		t.Errorf("original stderr = %q, want %q", got, want)
	}

	// #3 -- path to output file includes subdirectories.
	cmd = helperCommand("echo", "I can handle subdirectories")

	names = append(names, filepath.Join("sub", "foo", "bar"))
	contents = append(contents, cmd.Args[len(cmd.Args)-1]+"\n")

	if err := dr.Run(cmd, names[len(names)-1]); err != nil {
		t.Errorf("Run(%q) = %v, want %v", cmd.Args[3:], err, nil)
	}
	if err := dirHasExactFiles(dir, names, contents); err != nil {
		t.Error(err)
	}

	// #4 -- path is empty returns error, nothing new written to disk.
	cmd = helperCommand("echo", "no path")

	if err := dr.Run(cmd, ""); err == nil || !strings.Contains(err.Error(), "missing path") {
		t.Errorf("Run(%q) = %v, want %v", cmd.Args[3:], err, nil)
	}
	if err := dirHasExactFiles(dir, names, contents); err != nil {
		t.Error(err)
	}
}

func tree(dir string) string {
	b, _ := exec.Command("tree", "-Fah", dir).CombinedOutput()
	return string(b)
}

func dirHasExactFiles(dir string, names, contents []string) error {
	gotNames, err := readDir(dir)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(gotNames, names) {
		return fmt.Errorf("names = %v, want %v", gotNames, names)
	}
	for i, name := range names {
		b, err := ioutil.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if string(b) != contents[i] {
			return fmt.Errorf("file %q = %q, want %q", name, b, contents[i])
		}
	}
	return nil
}

func readDir(dir string) ([]string, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(fis))
	for _, fi := range fis {
		if fi.IsDir() {
			subDir := fi.Name()
			subNames, err := readDir(filepath.Join(dir, subDir))
			if err != nil {
				return nil, err
			}
			for _, name := range subNames {
				names = append(names, filepath.Join(subDir, name))
			}
			continue
		}
		names = append(names, fi.Name())
	}
	return names, nil
}
