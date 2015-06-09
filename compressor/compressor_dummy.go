package compressor

import (
	"io"

	"github.com/CIRCL/pbtc/adaptor"
)

// CompressorDummy is an empty compressor implementing the compressor
// interface. It can be used in place of other compressors to provide
// uncompressed input and output.
type CompressorDummy struct {
	Compressor
}

// NewDummy creates a new dummy compressor which does not compress output or
// decompress input. It can serve as a placeholder or for debugging.
func NewDummy(options ...func(adaptor.Compressor)) *CompressorDummy {
	comp := &CompressorDummy{}

	for _, option := range options {
		option(comp)
	}

	return comp
}

// GetWriter simply returns the original writer to the caller, so as not to
// affect the written data at all.
func (comp *CompressorDummy) GetWriter(writer io.Writer) (io.Writer, error) {
	return writer, nil
}

// GetReader returns the original reader to the caller, so as not to affect the
// read data at all.
func (comp *CompressorDummy) GetReader(reader io.Reader) (io.Reader, error) {
	return reader, nil
}
