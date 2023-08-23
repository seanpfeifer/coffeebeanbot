package coffeebeanbot

import "log/slog"

// LogIfError will log the [error + args as key:value pairs] if the error is non-nil.
// Returns true if err is non-nil.
func LogIfError(logger *slog.Logger, err error, msg string, args ...any) bool {
	if err != nil {
		kv := append([]any{"error", err}, args...)
		logger.Error(msg, kv...)
		return true
	}
	return false
}
