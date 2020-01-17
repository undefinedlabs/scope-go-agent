// +build !go1.13

package logging

import (
	"io"
	"os"
)

// Gets the standard logger writer
func getStdLoggerWriter() io.Writer {
	return os.Stderr // There is no way to get the current writer for the standard logger, but the default one is os.Stderr
}
