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

func TestConfigHash_UsesLengthDelimitedEncoding(t *testing.T) {
	configA := &Config{
		Scheme:     "s3://",
		BucketName: "abc",
		Prefix:     "def/",
	}
	configB := &Config{
		Scheme:     "s3://",
		BucketName: "abcd",
		Prefix:     "ef/",
	}

	require.NotEqual(t, configA.Hash(), configB.Hash())
}

func TestSplitObjectURI_RejectsEncodedQueryDelimitersInPath(t *testing.T) {
	_, _, err := SplitObjectURI("s3://bucket/other/%3Fendpoint=attacker.example:9000%26disableSSL=true/file")
	require.Error(t, err)
	require.Contains(t, err.Error(), "encoded query delimiters")
}

func TestSplitObjectURI_PreservesDecodedObjectKeys(t *testing.T) {
	testCases := []struct {
		name       string
		uri        string
		wantPrefix string
		wantBase   string
	}{
		{
			name:       "space",
			uri:        "s3://bucket/path/my%20model",
			wantPrefix: "s3://bucket/path",
			wantBase:   "my model",
		},
		{
			name:       "percent",
			uri:        "s3://bucket/path/100%25complete",
			wantPrefix: "s3://bucket/path",
			wantBase:   "100%complete",
		},
		{
			name:       "unicode",
			uri:        "s3://bucket/path/caf%C3%A9",
			wantPrefix: "s3://bucket/path",
			wantBase:   "café",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prefix, base, err := SplitObjectURI(testCase.uri)
			require.NoError(t, err)
			require.Equal(t, testCase.wantPrefix, prefix)
			require.Equal(t, testCase.wantBase, base)
		})
	}
}

func TestParseBucketPathToConfig_RejectsEncodedQueryDelimitersInPath(t *testing.T) {
	_, err := ParseBucketPathToConfig("s3://bucket/other/%3Fendpoint=attacker.example:9000%26disableSSL=true/")
	require.Error(t, err)
	require.Contains(t, err.Error(), "encoded query delimiters")
}
