package reflection

import (
	"errors"
	"reflect"
	"unsafe"
)

// Gets the type pointer to a field name
func GetTypePointer(i interface{}, fieldName string) (reflect.Type, error) {
	typeOf := reflect.Indirect(reflect.ValueOf(i)).Type()
	if member, ok := typeOf.FieldByName(fieldName); ok {
		return reflect.PtrTo(member.Type), nil
	}
	return nil, errors.New("field can't be retrieved")
}

// Gets a pointer of a private or public field in any struct
func GetFieldPointerOf(i interface{}, fieldName string) (unsafe.Pointer, error) {
	val := reflect.Indirect(reflect.ValueOf(i))
	member := val.FieldByName(fieldName)
	if member.IsValid() {
		ptrToY := unsafe.Pointer(member.UnsafeAddr())
		return ptrToY, nil
	}
	return nil, errors.New("field can't be retrieved")
}
