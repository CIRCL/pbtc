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
