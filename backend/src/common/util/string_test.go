package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	assert.Equal(t, Truncate("123456", 10), "123456")
	assert.Equal(t, Truncate("123456", 2), "12")
	assert.Equal(t, Truncate("", 10), "")
	assert.Equal(t, Truncate("123456", 0), "")
}
