package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtrToStringOrDefault(t *testing.T) {
	val := "string"
	ptr := &val

	result1 := PtrToStringOrDefault(ptr, val)
	assert.Equal(t, "string", result1)

	result2 := PtrToStringOrDefault(nil, "default")
	assert.Equal(t, "default", result2)
}

func TestStringToPtr(t *testing.T) {
	str := "non-empty"
	ptr := &str

	result1 := StringToPtr(str)
	assert.NotNil(t, result1)
	assert.Equal(t, result1, ptr)
	assert.Equal(t, "non-empty", *result1)

	result2 := StringToPtr("")
	assert.Nil(t, result2)
}
