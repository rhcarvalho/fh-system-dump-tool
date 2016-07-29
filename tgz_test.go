package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestWritingAFile(t *testing.T) {
	b := bytes.Buffer{}
	tgzFile := bufio.NewReadWriter(bufio.NewReader(&b), bufio.NewWriter(&b))

	tgz, err := NewTgz(tgzFile)
	if err != nil {
		t.Fatal(err)
	}

	writer := tgz.GetWriterToFile("test1.txt")
	writer.Write([]byte("test"))
	writer.Close()
	tgz.Close()
	tgzFile.Flush()

	files, err := decompressAndListFiles(tgzFile)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := files["test1.txt"]; !ok {
		t.Fatal("Expected tgz to contain test1.txt but it didnt")
	}
}

func decompressAndListFiles(tgzFile io.Reader) (map[string]string, error) {
	ret := map[string]string{}

	gzf, err := gzip.NewReader(tgzFile)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzf)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		ret[header.Name] = header.Name
	}

	return ret, nil
}
