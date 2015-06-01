package compressor

import (
	"io"

	lz4 "github.com/pwaller/go-clz4"

	"github.com/CIRCL/pbtc/adaptor"
)

// CompressorLZ4 is a wrapper around the LZ4 compression library fulfilling the
// compressor interface. This allows us to create LZ4 readers and writers at
// runtime.
type CompressorLZ4 struct {
	log adaptor.Log
}

// NewLZ4 creates a new wrapper around the LZ4 compression library.
func NewLZ4(options ...func(*CompressorLZ4)) *CompressorLZ4 {
	comp := &CompressorLZ4{}

	for _, option := range options {
		option(comp)
	}

	return comp
}

// Close shuts the compressor down.
func (comp *CompressorLZ4) Close() {
}

// GetWriter wraps a new LZ4 writer around the provided writer and returns it
// as an interface value. This allows us to have a common function signature
// with other compression libraries.
func (comp *CompressorLZ4) GetWriter(writer io.Writer) (io.Writer, error) {
	return lz4.NewWriter(writer), nil
}

// GetReader wraps a new LZ4 reader around the provided reader and returns it
// as an interface value. This allows us to have a common function signature
// with other compression libraries.
func (comp *CompressorLZ4) GetReader(reader io.Reader) (io.Reader, error) {
	return lz4.NewReader(reader), nil
}
