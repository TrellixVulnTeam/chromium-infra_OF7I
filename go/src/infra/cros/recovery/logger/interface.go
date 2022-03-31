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
}

// LogIndenter represents a simple interface to provide option to set indent logs.
// Interface mostly used for local debugging and testing.
type LogIndenter interface {
	// Indent increment indentation for logger.
	Indent()
	// Dedent decrement indentation for logger.
	Dedent()
}
