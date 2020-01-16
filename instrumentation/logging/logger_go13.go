// +build "go1.13"

package logging

import stdlog "log"

// Patch the standard logger
func PatchStandardLogger() {
	oldLoggerWriter = stdlog.Writer()
	loggerWriter := newInstrumentedWriter(oldLoggerWriter, stdlog.Prefix(), stdlog.Flags())
	stdlog.SetOutput(loggerWriter)
	otWriters = append(otWriters, loggerWriter)
}
