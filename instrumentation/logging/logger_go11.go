// +build !go1.12

package logging

import (
	"errors"
	"io"
	stdlog "log"
	"os"
	"reflect"
	"unsafe"
)

// Gets the standard logger writer
func getStdLoggerWriter() io.Writer {
	return os.Stderr // There is no way to get the current writer for the standard logger, but the default one is os.Stderr
}

// Gets the writer of a custom logger
func getLoggerWriter(logger *stdlog.Logger) io.Writer {
	// There is not API in Go1.11 to get the current writer, accessing by reflection.
	if ptr, err := GetFieldPointerOfLogger(logger, "out"); err == nil {
		return *(*io.Writer)(ptr)
	}
	return nil
}

// Gets a pointer of a private or public field in a testing.T struct
func GetFieldPointerOfLogger(logger *stdlog.Logger, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(logger))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}
