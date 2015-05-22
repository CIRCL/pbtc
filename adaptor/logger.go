package adaptor

type Logger interface {
	SetLevel(module string, level int)
	GetLog(module string) Log
}
