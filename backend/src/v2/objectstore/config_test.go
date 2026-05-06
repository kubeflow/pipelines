package objectstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseBucketPathToConfig(t *testing.T) {
	tests := []struct {
		name               string
		pipelineRoot       string
		expectedScheme     string
		expectedPrefix     string
		expectedBucketName string
	}{
		{
			name:               "s3 path",
			pipelineRoot:       "s3://mlpipeline/v2/artifacts/",
			expectedScheme:     "s3://",
			expectedPrefix:     "v2/artifacts/",
			expectedBucketName: "mlpipeline",
		},
		{
			name:               "file path",
			pipelineRoot:       "file:///tmp/kfp-artifacts/v2/artifacts/",
			expectedScheme:     "file:///",
			expectedPrefix:     "kfp-artifacts/v2/artifacts/",
			expectedBucketName: "tmp",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseBucketPathToConfig(test.pipelineRoot)
			require.NoError(t, err)
			require.Equal(t, test.expectedScheme, result.Scheme)
			require.Equal(t, test.expectedPrefix, result.Prefix)
			require.Equal(t, test.expectedBucketName, result.BucketName)
		})
	}
}

func TestParseProviderFromPath_File(t *testing.T) {
	provider, err := ParseProviderFromPath("file:///tmp/kfp-artifacts/v2/artifacts/")
	require.NoError(t, err)
	require.Equal(t, "file", provider)
}
