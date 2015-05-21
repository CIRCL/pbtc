package adaptor

import (
	"io"
)

type Compressor interface {
	GetWriter(io.Writer) (io.Writer, error)
	GetReader(io.Reader) (io.Reader, error)
}
