package clientmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedTable(t *testing.T) {
	assert.True(t, isAllowedTable("pipeline"))
	assert.False(t, isAllowedTable("users"))
}
