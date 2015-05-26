package adaptor

// Record defines a common interface for records that describe an event on the
// Bitcoin network. They provide the output in string and binary format at this
// point.
type Record interface {
	String() string
	Bytes() []byte
}
