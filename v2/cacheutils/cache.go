package cacheutils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
)

type CacheKey struct {
	inputArtifactNames   map[string]artifactNameList
	inputParameters      map[string]pipeline_spec.Value
	outputArtifactsSpec  map[string]pipeline_spec.RuntimeArtifact
	outputParametersSpec map[string]string
	containerSpec        containerSpec
}

type artifactNameList struct {
	// A list of artifact Names.
	artifactNames []string
}

type containerSpec struct {
	image   string
	cmdArgs []string
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

func GenerateCacheKey(
	inputs *pipeline_spec.ExecutorInput_Inputs,
	outputs *pipeline_spec.ExecutorInput_Outputs,
	outputParametersTypeMap map[string]string,
	cmdArgs []string, image string) (*CacheKey, error) {

	cacheKey := CacheKey{
		inputArtifactNames:   make(map[string]artifactNameList),
		inputParameters:      make(map[string]pipeline_spec.Value),
		outputArtifactsSpec:  make(map[string]pipeline_spec.RuntimeArtifact),
		outputParametersSpec: make(map[string]string),
	}

	for inputArtifactName, inputArtifactList := range inputs.GetArtifacts() {
		artifactNameList := artifactNameList{artifactNames: make([]string, 0)}
		for _, artifact := range inputArtifactList.Artifacts {
			artifactNameList.artifactNames = append(artifactNameList.artifactNames, artifact.GetName())
		}
		cacheKey.inputArtifactNames[inputArtifactName] = artifactNameList
	}

	for inputParameterName, inputParameterValue := range inputs.GetParameters() {
		cacheKey.inputParameters[inputParameterName] = pipeline_spec.Value{
			Value: inputParameterValue.Value,
		}
	}

	for outputArtifactName, outputArtifactList := range outputs.GetArtifacts() {
		if len(outputArtifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		outputArtifact := outputArtifactList.Artifacts[0]
		outputArtifactWithUriWiped := pipeline_spec.RuntimeArtifact{
			Name:     outputArtifact.GetName(),
			Type:     outputArtifact.GetType(),
			Metadata: outputArtifact.GetMetadata(),
		}
		cacheKey.outputArtifactsSpec[outputArtifactName] = outputArtifactWithUriWiped
	}

	for outputParameterName, _ := range outputs.GetParameters() {
		outputParameterType, ok := outputParametersTypeMap[outputParameterName]
		if !ok {
			return nil, fmt.Errorf("unknown parameter %q found in ExecutorInput_Outputs", outputParameterName)
		}

		cacheKey.outputParametersSpec[outputParameterName] = outputParameterType
	}

	cacheKey.containerSpec = containerSpec{
		image:   image,
		cmdArgs: cmdArgs,
	}

	return &cacheKey, nil

}