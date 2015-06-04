package adaptor

import (
	"io"
)

// Compressor defines a common interface for compression & decompression
// algorithms. Due to the different nature of compression algorithms, this
// interface allows us to wrap around all kinds of compressors, including those
// that need to write headers or trailers.
type Compressor interface {
	GetWriter(io.Writer) (io.Writer, error)
	GetReader(io.Reader) (io.Reader, error)
}
