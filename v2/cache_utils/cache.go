package cache_utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
)

type CacheKey struct {
	InputArtifactNames   map[string]ArtifactNameList
	InputParameters      map[string]pipeline_spec.Value
	OutputArtifactsSpec  map[string]pipeline_spec.RuntimeArtifact
	OutputParametersSpec map[string]string
	ContainerSpec        ContainerSpec
}

type ArtifactNameList struct {
	// A list of artifact Names.
	ArtifactNames []string
}

type ContainerSpec struct {
	Image   string
	CmdArgs []string
}

func GenerateFingerPrint(cacheKey CacheKey) (string, error) {
	b, err := json.Marshal(cacheKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key: %w", err)
	}
	hash := sha256.New()
	hash.Write(b)
	md := hash.Sum(nil)
	executionHashKey := hex.EncodeToString(md)
	return executionHashKey, nil

}
