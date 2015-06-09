package compressor

import (
	"io"

	lz4 "github.com/pwaller/go-clz4"

	"github.com/CIRCL/pbtc/adaptor"
)

// CompressorLZ4 is a wrapper around the LZ4 compression library implementing
// the compressor interface. This allows us to create LZ4 readers and writers at
// runtime.
type CompressorLZ4 struct {
	Compressor
}

// NewLZ4 creates a new wrapper around the LZ4 compression library.
func NewLZ4(options ...func(adaptor.Compressor)) *CompressorLZ4 {
	comp := &CompressorLZ4{}

	for _, option := range options {
		option(comp)
	}

	return comp
}

// GetWriter wraps a new LZ4 writer around the provided writer.
func (comp *CompressorLZ4) GetWriter(writer io.Writer) (io.Writer, error) {
	return lz4.NewWriter(writer), nil
}

// GetReader wraps a new LZ4 reader around the provided reader.
func (comp *CompressorLZ4) GetReader(reader io.Reader) (io.Reader, error) {
	return lz4.NewReader(reader), nil
}
