package adaptor

import (
	"io"
)

// Compressor defines a common interface for compression & decompression
// algorithms to be used by the recorder. Due to an inflexibility in the Go
// function types, we have to wrap them in this.
type Compressor interface {
	GetWriter(io.Writer) (io.Writer, error)
	GetReader(io.Reader) (io.Reader, error)
}
