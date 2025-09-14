package fftx

import (
	"errors"
	"io"
	"os"
)

type File struct {
	path   string
	file   *os.File
	writer *io.OffsetWriter
	reader *io.SectionReader
	size   int64
}

func openReadFile(path string) (*os.File, error) {
	var err error
	var file *os.File
	file, err = os.OpenFile(path, 0, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func openOrCreateFile(path string) (*os.File, error) {
	var err error
	var file *os.File
	file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.Create(path)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	return file, nil
}
