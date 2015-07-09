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

package processor

import (
	"errors"

	"github.com/CIRCL/pbtc/adaptor"
)

type ProcessorType int

const (
	AddressFilterType ProcessorType = iota
	CommandFilterType
	IPFilterType
	FileWriterType
	RedisWriterType
	ZeroMQWriterType
)

func ParseType(processor string) (ProcessorType, error) {
	switch processor {
	case "ADDRESS_FILTER":
		return AddressFilterType, nil

	case "COMMAND_FILTER":
		return CommandFilterType, nil

	case "IP_FILTER":
		return IPFilterType, nil

	case "FILE_WRITER":
		return FileWriterType, nil

	case "REDIS_WRITER":
		return RedisWriterType, nil

	case "ZEROMQ_WRITER":
		return ZeroMQWriterType, nil

	default:
		return -1, errors.New("invalid processor string")
	}
}

// New returns a new default filter.
func New() (adaptor.Processor, error) {
	return NewDummy()
}

type Processor struct {
	log  adaptor.Log
	next []adaptor.Processor
}

func (pro *Processor) SetLog(log adaptor.Log) {
	pro.log = log
}

func (pro *Processor) AddNext(next adaptor.Processor) {
	pro.next = append(pro.next, next)
}
