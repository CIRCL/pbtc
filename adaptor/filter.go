package adaptor

// Filter defines an interface for filters to work on messages from the Bitcoin
// network. It will filter the messages according to a number of criteria
// before forwarding them to the added writers.
type Processor interface {
	Process(record Record)
}
