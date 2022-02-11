package coffeebeanbot

// Logger is the interface that all logs are expected to use with this package.
type Logger interface {
	// Info logs a message with some additional context as key-value pairs
	Info(msg string, kvPairs ...any)
	// Error logs a message with some additional context as key-value pairs
	Error(msg string, kvPairs ...any)
	// Named adds a sub-scope to the logger's name
	Named(name string) Logger
}

// LogIfError will log the [error + extra info] if the error is non-nil.
// Returns true if err is non-nil.
func LogIfError(logger Logger, err error, msg string, extraInfo ...any) bool {
	if err != nil {
		info := append([]any{"error", err}, extraInfo...)
		logger.Error(msg, info...)
		return true
	}
	return false
}
