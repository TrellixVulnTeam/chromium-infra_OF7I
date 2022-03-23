package logger

// Logger represents a simple interface for logging data.
type Logger interface {
	// Debugf log message at Debugf level.
	Debugf(format string, args ...interface{})
	// Infof is like Debug, but logs at Infof level.
	Infof(format string, args ...interface{})
	// Warningf is like Debug, but logs at Warningf level.
	Warningf(format string, args ...interface{})
	// Errorf is like Debug, but logs at Errorf level.
	Errorf(format string, args ...interface{})
	// IndentLogging increment indentation for logger.
	IndentLogging()
	// DedentLogging decrement indentation for logger.
	DedentLogging()
}
