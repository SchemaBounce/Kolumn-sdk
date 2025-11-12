package telemetry

import "sync"

// Factory creates telemetry loggers scoped to a component name.
type Factory interface {
	New(component string) Logger
}

// FactoryFunc adapts a function into a telemetry logger factory.
type FactoryFunc func(component string) Logger

// New implements Factory.
func (f FactoryFunc) New(component string) Logger {
	return f(component)
}

var (
	factoryMu     sync.RWMutex
	loggerFactory Factory = defaultFactory{}
)

// SetLoggerFactory installs a custom logger factory. Passing nil restores the
// built-in structured logger factory.
func SetLoggerFactory(factory Factory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()

	if factory == nil {
		loggerFactory = defaultFactory{}
		return
	}
	loggerFactory = factory
}

// ResetLoggerFactory resets the telemetry logger factory back to the default.
func ResetLoggerFactory() {
	SetLoggerFactory(nil)
}

func currentFactory() Factory {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	return loggerFactory
}

type defaultFactory struct{}

func (defaultFactory) New(component string) Logger {
	return newStructuredLogger(component)
}
