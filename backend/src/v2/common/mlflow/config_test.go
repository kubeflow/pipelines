package mlflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutionStateToMLflowTerminalStatus(t *testing.T) {
	assert.Equal(t, "FINISHED", ExecutionStateToMLflowTerminalStatus("COMPLETE"))
	assert.Equal(t, "FINISHED", ExecutionStateToMLflowTerminalStatus("CACHED"))
	assert.Equal(t, "KILLED", ExecutionStateToMLflowTerminalStatus("CANCELED"))
	assert.Equal(t, "FAILED", ExecutionStateToMLflowTerminalStatus("FAILED"))
	assert.Equal(t, "FAILED", ExecutionStateToMLflowTerminalStatus("UNKNOWN"))
}
