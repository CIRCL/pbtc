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

package logger

import (
	"github.com/op/go-logging"

	"github.com/CIRCL/pbtc/adaptor"
)

var logr adaptor.Logger
var err error

func init() {
	gologr, goerr := NewGologging(
		SetConsoleEnabled(true),
		SetConsoleLevel(logging.DEBUG),
	)

	logr, err = gologr, goerr
}

// New returns the default logger for the package. Use this to define default
// settings and library.
func New() (adaptor.Logger, error) {
	return logr, err
}
