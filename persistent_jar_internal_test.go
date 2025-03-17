package cookiejar

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntry_HasSameProps(t *testing.T) {
	t.Parallel()

	exported := make(map[string]reflect.Type)
	typeOfExported := reflect.TypeOf(Entry{})

	for i := range typeOfExported.NumField() {
		field := typeOfExported.Field(i)
		exported[field.Name] = field.Type
	}

	unexported := make(map[string]reflect.Type)
	typeOfUnexported := reflect.TypeOf(entry{})

	for i := range typeOfUnexported.NumField() {
		field := typeOfUnexported.Field(i)
		if field.Name == "seqNum" {
			field.Name = "SeqNum"
		}

		unexported[field.Name] = field.Type
	}

	assert.Equalf(t, exported, unexported, "exported and unexported entries must be the same")
}
