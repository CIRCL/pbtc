// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

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
