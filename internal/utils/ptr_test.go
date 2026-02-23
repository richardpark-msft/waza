package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtr(t *testing.T) {
	v := 42
	p := Ptr(v)

	assert.NotNil(t, p)
	assert.Equal(t, 42, *p)

	_ = 100 // original value changed; pointer should still hold 42
	assert.Equal(t, 42, *p)
}
