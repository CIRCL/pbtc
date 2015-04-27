package adaptor

type Record interface {
	String() string
	Bytes() []byte
}
