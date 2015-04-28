package recorder

type Record interface {
	String() string
	Bytes() []byte
}
