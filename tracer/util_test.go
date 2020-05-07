package tracer

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUIDStringConverter(t *testing.T) {
	for i := 0; i < 10000; i++ {
		id := uuid.New()
		val := UUIDToString(id)
		idRes, err := StringToUUID(val)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, id.String(), idRes.String())
	}
}
