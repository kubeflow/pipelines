package util

import (
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/require"
)

func TestOutputIOTypeForIteration(t *testing.T) {
	require.Equal(t, apiv2.IOType_OUTPUT, OutputIOTypeForIteration(nil))
	index := int64(0)
	require.Equal(t, apiv2.IOType_ITERATOR_OUTPUT, OutputIOTypeForIteration(&index))
}
