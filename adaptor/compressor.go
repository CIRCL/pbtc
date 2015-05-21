package adaptor

import (
	"io"
)

type Compressor interface {
	GetWriter(io.Writer) io.Writer
	GetReader(io.Reader) io.Reader
}
