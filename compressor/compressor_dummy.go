package compressor

import (
	"io"
)

// CompressorDummy is a wrapper around the LZ4 compression library, allowing
// run-time creation of readers and writers as interface values.
type CompressorDummy struct{}

// NewLZ4 creates a new wrapper around the LZ4 compression library.
func NewDummy() *CompressorDummy {
	comp := &CompressorDummy{}

	return comp
}

// GetWriter wraps a new LZ4 writer around the provided writer and returns it
// as an interface value. This allows us to have a common function signature
// with other compression libraries.
func (comp *CompressorDummy) GetWriter(writer io.Writer) (io.Writer, error) {
	return io.MultiWriter(writer), nil
}

// GetReader wraps a new LZ4 reader around the provided reader and returns it
// as an interface value. This allows us to have a common function signature
// with other compression libraries.
func (comp *CompressorDummy) GetReader(reader io.Reader) (io.Reader, error) {
	return io.MultiReader(reader), nil
}
