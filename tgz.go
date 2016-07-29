package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"time"
)

type Archive struct {
	tgzFile   io.Writer
	tarWriter *tar.Writer
	gzWriter  *gzip.Writer
}

type ArchiveWriter struct {
	File    string
	Archive *Archive
	Writer  *bytes.Buffer
}

func (a *ArchiveWriter) Write(p []byte) (n int, err error) {
	return a.Writer.Write(p)
}

func (a *ArchiveWriter) Close() error {
	return a.Archive.AddFileByContent(a.Writer.Bytes(), a.File)
}

func NewTgz(file io.Writer) (*Archive, error) {
	tgz := Archive{}
	var err error
	tgz.tgzFile = file
	if err != nil {
		return nil, err
	}

	tgz.gzWriter = gzip.NewWriter(tgz.tgzFile)
	tgz.tarWriter = tar.NewWriter(tgz.gzWriter)

	return &tgz, nil
}

func (a *Archive) GetWriterToFile(file string) io.WriteCloser {
	writer := ArchiveWriter{File: file, Archive: a, Writer: &bytes.Buffer{}}
	return &writer
}

func (a *Archive) AddFileByContent(src []byte, dest string) error {
	header := &tar.Header{
		Name:    dest,
		Size:    int64(len(src)),
		Mode:    0775,
		ModTime: time.Now(),
	}

	if err := a.tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(a.tarWriter, bytes.NewReader(src)); err != nil {
		return err
	}

	return nil
}

func (a *Archive) Close() {
	a.tarWriter.Close()
	a.gzWriter.Close()
}
