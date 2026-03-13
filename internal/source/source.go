package source

import (
	"io"
	"os"
)

type KeywordSource interface {
	Open() (io.ReadCloser, error)
	Cleanup() error
}
type CSVFileSource struct {
	Path string
}

func (s *CSVFileSource) Open() (io.ReadCloser, error) {
	return os.Open(s.Path)
}

func (s *CSVFileSource) Cleanup() error {
	return os.Remove(s.Path)
}
