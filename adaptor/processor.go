package adaptor

// Processor defines the interface for any structure that will process records
// received from the Bitcoin network. It is currently used for filters, which
// will forward them to other processors, and writers, which will take the
// records and output them to certain media.
type Processor interface {
	SetNext(...Processor)
	Process(Record)
}
