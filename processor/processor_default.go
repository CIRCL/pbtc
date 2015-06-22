package processor

import (
	"errors"

	"github.com/CIRCL/pbtc/adaptor"
)

type ProcessorType int

const (
	Base58F ProcessorType = iota
	CommandF
	IPF
	FileW
	RedisW
	ZeroMQW
)

func ParseType(processor string) (ProcessorType, error) {
	switch processor {
	case "Base58Filter":
		return Base58F, nil

	case "CommandFilter":
		return CommandF, nil

	case "IPFilter":
		return IPF, nil

	case "FileWriter":
		return FileW, nil

	case "RedisWriter":
		return RedisW, nil

	case "ZeroMQWriter":
		return ZeroMQW, nil

	default:
		return -1, errors.New("invalid processor string")
	}
}

// New returns a new default filter.
func New() (adaptor.Processor, error) {
	return NewDummy()
}

func SetNext(next ...adaptor.Processor) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		pro.SetNext(next...)
	}
}

type Processor struct {
	log  adaptor.Log
	next []adaptor.Processor
}

func (pro *Processor) SetNext(next ...adaptor.Processor) {
	pro.next = next
}

func (pro *Processor) SetLog(log adaptor.Log) {
	pro.log = log
}
