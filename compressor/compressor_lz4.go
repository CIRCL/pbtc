package compressor

import (
	"io"

	"github.com/pierrec/lz4"
)

type CompressorLZ4 struct{}

func NewCompressorLZ4() *CompressorLZ4 {
	comp := &CompressorLZ4{}

	return comp
}

func (comp *CompressorLZ4) GetWriter(writer io.Writer) (io.Writer, error) {
	return lz4.NewWriter(writer), nil
}

func (comp *CompressorLZ4) GetReader(reader io.Reader) (io.Reader, error) {
	return lz4.NewReader(reader), nil
}
