package adaptor

// Writer defines an output channel for logs.
type Writer interface {
	Line(string)
}
