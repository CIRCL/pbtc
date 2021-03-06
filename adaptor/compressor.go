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

package adaptor

import (
	"io"
)

// Compressor defines a common interface for compression & decompression
// algorithms. Due to the different nature of compression algorithms, this
// interface allows us to wrap around all kinds of compressors, including those
// that need to write headers or trailers.
type Compressor interface {
	SetLog(Log)
	GetWriter(io.Writer) (io.Writer, error)
	GetReader(io.Reader) (io.Reader, error)
}
