package objectstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseBucketPathToConfig(t *testing.T) {
	pipelineRoot := "s3://mlpipeline/v2/artifacts/"
	result, err := ParseBucketPathToConfig(pipelineRoot)
	require.NoError(t, err)
	require.Equal(t, "s3://", result.Scheme)
	require.Equal(t, "v2/artifacts/", result.Prefix)
	require.Equal(t, "mlpipeline", result.BucketName)
}
