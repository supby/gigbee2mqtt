package reflector

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetInterfaceValue(t *testing.T) {

	var valueToSet float64
	valueToSet = 200

	ret := ConvertType(valueToSet, reflect.Uint16)

	assert.Equal(t, uint16(200), ret)

}
