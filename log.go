package coffeebeanbot

// Logger is the interface that all logs are expected to use with this package.
type Logger interface {
	// Info logs a message with some additional context as key-value pairs
	Info(msg string, kvPairs ...interface{})
	// Error logs a message with some additional context as key-value pairs
	Error(msg string, kvPairs ...interface{})
	// Named adds a sub-scope to the logger's name
	Named(name string) Logger
}
