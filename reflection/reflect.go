package reflection

import (
	"errors"
	stdlog "log"
	"reflect"
	"testing"
	"unsafe"
)

// Gets a pointer of a private or public field in a testing.M struct
func GetFieldPointerOfM(m *testing.M, fieldName string) (unsafe.Pointer, error) {
	return getFieldPointerOf(m, fieldName)
}

// Gets a pointer of a private or public field in a testing.T struct
func GetFieldPointerOfT(t *testing.T, fieldName string) (unsafe.Pointer, error) {
	return getFieldPointerOf(t, fieldName)
}

// Gets a pointer of a private or public field in a testing.B struct
func GetFieldPointerOfB(b *testing.B, fieldName string) (unsafe.Pointer, error) {
	return getFieldPointerOf(b, fieldName)
}

// Gets a pointer of a private or public field in a testing.T struct
func GetFieldPointerOfLogger(logger *stdlog.Logger, fieldName string) (unsafe.Pointer, error) {
	return getFieldPointerOf(logger, fieldName)
}

// Gets the type pointer to a field name
func GetTypePointer(i interface{}, fieldName string) (reflect.Type, error) {
	typeOf := reflect.Indirect(reflect.ValueOf(i)).Type()
	if member, ok := typeOf.FieldByName(fieldName); ok {
		return reflect.PtrTo(member.Type), nil
	}
	return nil, errors.New("field can't be retrieved")
}

func getFieldPointerOf(i interface{}, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(i))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}
