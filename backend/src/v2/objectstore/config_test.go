// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package objectstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasStructuredS3Settings(t *testing.T) {
	assert.False(t, HasStructuredS3Settings(map[string]string{}))

	assert.False(t, HasStructuredS3Settings(map[string]string{
		S3ParamFromEnv:      "true",
		S3ParamSecretName:   "secret",
		S3ParamAccessKeyKey: "access",
		S3ParamSecretKeyKey: "key",
	}))

	assert.True(t, HasStructuredS3Settings(map[string]string{
		S3ParamFromEnv: "true",
		S3ParamRegion:  "us-east-1",
	}))

	for _, key := range []string{
		S3ParamRegion,
		S3ParamEndpoint,
		S3ParamDisableSSL,
		S3ParamForcePathStyle,
		S3ParamMaxRetries,
	} {
		t.Run(key, func(t *testing.T) {
			assert.True(t, HasStructuredS3Settings(map[string]string{key: "x"}))
		})
	}
}

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
		{
			name:       "alternate ASCII escape",
			uri:        "s3://bucket/path/discount%50off",
			wantPrefix: "s3://bucket/path",
			wantBase:   "discountPoff",
		},
		{
			name:       "lowercase Unicode escapes",
			uri:        "s3://bucket/path/caf%c3%a9",
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

func TestSplitObjectURI_RejectsMalformedRawPercent(t *testing.T) {
	_, _, err := SplitObjectURI("s3://bucket/path/100%complete")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid URL escape")
}

func TestParseBucketPathToConfig_RejectsEncodedQueryDelimitersInPath(t *testing.T) {
	_, err := ParseBucketPathToConfig("s3://bucket/other/%3Fendpoint=attacker.example:9000%26disableSSL=true/")
	require.Error(t, err)
	require.Contains(t, err.Error(), "encoded query delimiters")
}

func TestValidatePipelineRoot_RejectsPercentInPrefix(t *testing.T) {
	for _, root := range []string{
		"s3://bucket/root%20dir",
		"s3://bucket/root%20dir/x",
		"gs://bucket/100%complete",
		"minio://bucket/root%25dir/sub",
	} {
		t.Run(root, func(t *testing.T) {
			err := ValidatePipelineRoot(root)
			require.Error(t, err)
			require.Contains(t, err.Error(), "percent signs are not allowed in pipeline root prefix")
			require.Contains(t, err.Error(), root)
		})
	}
}

func TestParseBucketPathToConfig_AllowsPercentInDecodedArtifactPrefix(t *testing.T) {
	// Imported artifact URIs are decoded by SplitObjectURI before being parsed,
	// so a valid artifact key such as "100%complete" must still be accepted.
	prefix, base, err := SplitObjectURI("s3://bucket/root/100%25complete/model")
	require.NoError(t, err)
	assert.Equal(t, "s3://bucket/root/100%complete", prefix)
	assert.Equal(t, "model", base)

	cfg, err := ParseBucketPathToConfig(prefix)
	require.NoError(t, err)
	assert.Equal(t, "root/100%complete/", cfg.Prefix)

	rootCfg, err := ParseBucketPathToConfig("s3://bucket/root")
	require.NoError(t, err)
	assert.True(t, IsWithinBucketRoot(rootCfg, cfg))
}

func TestParseBucketPathToConfig_AllowsSpacesUnicodeAmpersandAndQuery(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		expectedPrefix string
		expectedQuery  string
	}{
		{
			name:           "spaces",
			path:           "s3://bucket/root dir/x",
			expectedPrefix: "root dir/x/",
		},
		{
			name:           "unicode",
			path:           "s3://bucket/café/日本語",
			expectedPrefix: "café/日本語/",
		},
		{
			name:           "ampersand in prefix",
			path:           "s3://bucket/a&b/x",
			expectedPrefix: "a&b/x/",
		},
		{
			name:           "query string with question mark and ampersand",
			path:           "s3://bucket/root dir?endpoint=seaweedfs:9000&disableSSL=true",
			expectedPrefix: "root dir/",
			expectedQuery:  "?endpoint=seaweedfs:9000&disableSSL=true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseBucketPathToConfig(tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedPrefix, cfg.Prefix)
			assert.Equal(t, tc.expectedQuery, cfg.QueryString)
		})
	}
}

func TestParseBucketPathToConfig_RegressionRootPercent20DirVsRootSpaceDir(t *testing.T) {
	// SplitObjectURI decodes %20, so an unescaped artifact URI built from a
	// "root%20dir" prefix would collide with "root dir/x".
	spacePrefix, spaceBase, err := SplitObjectURI("s3://bucket/root dir/x")
	require.NoError(t, err)
	escapedPrefix, escapedBase, err := SplitObjectURI("s3://bucket/root%20dir/x")
	require.NoError(t, err)
	assert.Equal(t, spacePrefix, escapedPrefix)
	assert.Equal(t, spaceBase, escapedBase)

	// "root dir" is accepted and round-trips through PrefixedBucket + SplitObjectURI.
	spaceRootCfg, err := ParseBucketPathToConfig("s3://bucket/root dir")
	require.NoError(t, err)
	persistedURI := spaceRootCfg.PrefixedBucket() + "/x"
	assert.Equal(t, "s3://bucket/root dir/x", persistedURI)
	resolvedPrefix, resolvedBase, err := SplitObjectURI(persistedURI)
	require.NoError(t, err)
	assert.Equal(t, "x", resolvedBase)
	resolvedPrefixCfg, err := ParseBucketPathToConfig(resolvedPrefix)
	require.NoError(t, err)
	assert.True(t, IsWithinBucketRoot(spaceRootCfg, resolvedPrefixCfg))

	// "root%20dir" and "root%20dir/x" are rejected as pipeline roots so the ambiguous pair cannot be configured.
	for _, badPath := range []string{"s3://bucket/root%20dir", "s3://bucket/root%20dir/x"} {
		err := ValidatePipelineRoot(badPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), badPath)
	}
}
