package adaptor

// Module defines the interface for all concurrently executing units to provide
// a setup method for initialization and a teardown function for shutdown.
type Module interface {
	Close()
}
