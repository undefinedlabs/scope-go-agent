package reflection

import (
	"errors"
	"reflect"
	"sync"
	"testing"
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

func GetTestMutex(t *testing.T) *sync.RWMutex {
	if ptr, err := GetFieldPointerOf(t, "mu"); err == nil {
		return (*sync.RWMutex)(ptr)
	}
	return nil
}

func GetIsParallel(t *testing.T) bool {
	mu := GetTestMutex(t)
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if pointer, err := GetFieldPointerOf(t, "isParallel"); err == nil {
		return *(*bool)(pointer)
	}
	return false
}
