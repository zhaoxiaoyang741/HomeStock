// Package-level adapter for third-party logger interfaces.

package logger

import (
	"fmt"
	"regexp"
)

// botTokenRe is kept for compatibility with the existing helper tests.
var botTokenRe = regexp.MustCompile(`(bot\d+:)([A-Za-z0-9_-]{4})[A-Za-z0-9_-]{12,}([A-Za-z0-9_-]{4})`)

// maskSecrets is intentionally not used in the adapter logging path for now.
func maskSecrets(s string) string {
	return botTokenRe.ReplaceAllString(s, "${1}${2}****${3}")
}

// Logger adapts common third-party logger-style methods to the package logger.
type Logger struct {
	component string
	levels    map[int]LogLevel
}

func (b *Logger) logArgs(level LogLevel, args ...any) {
	logMessage(level, b.component, fmt.Sprint(args...), nil)
}

func (b *Logger) logf(level LogLevel, format string, args ...any) {
	logMessage(level, b.component, fmt.Sprintf(format, args...), nil)
}

func (b *Logger) resolveLevel(level int) LogLevel {
	if b.levels != nil {
		if mapped, ok := b.levels[level]; ok {
			return mapped
		}
	}

	return LogLevel(level)
}

// Debug logs debug messages.
func (b *Logger) Debug(v ...any) {
	b.logArgs(DEBUG, v...)
}

// Info logs info messages.
func (b *Logger) Info(v ...any) {
	b.logArgs(INFO, v...)
}

// Warn logs warning messages.
func (b *Logger) Warn(v ...any) {
	b.logArgs(WARN, v...)
}

// Error logs error messages.
func (b *Logger) Error(v ...any) {
	b.logArgs(ERROR, v...)
}

// Debugf logs formatted debug messages.
func (b *Logger) Debugf(format string, v ...any) {
	b.logf(DEBUG, format, v...)
}

// Infof logs formatted info messages.
func (b *Logger) Infof(format string, v ...any) {
	b.logf(INFO, format, v...)
}

// Warnf logs formatted warning messages.
func (b *Logger) Warnf(format string, v ...any) {
	b.logf(WARN, format, v...)
}

// Warningf logs formatted warning messages.
func (b *Logger) Warningf(format string, v ...any) {
	b.logf(WARN, format, v...)
}

// Errorf logs formatted error messages.
func (b *Logger) Errorf(format string, v ...any) {
	b.logf(ERROR, format, v...)
}

// Fatalf logs formatted fatal messages and exits.
func (b *Logger) Fatalf(format string, v ...any) {
	b.logf(FATAL, format, v...)
}

// Log logs a message at a given level with caller information
// the func name must be this because 3rd party loggers expect this
// msgL: message level (DEBUG, INFO, WARN, ERROR, FATAL)
// caller: unused parameter reserved for compatibility
// format: format string
// a: format arguments
//
//nolint:goprintffuncname
func (b *Logger) Log(msgL, _ int, format string, a ...any) {
	b.logf(b.resolveLevel(msgL), format, a...)
}

// Sync flushes log buffer (no-op for this implementation).
func (b *Logger) Sync() error {
	return nil
}

// WithLevels sets log levels mapping for this logger.
func (b *Logger) WithLevels(levels map[int]LogLevel) *Logger {
	b.levels = levels
	return b
}

// NewLogger creates a new logger instance with optional component name.
func NewLogger(component string) *Logger {
	return &Logger{component: component}
}
