package cacheutils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/cachekey"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
)

func GenerateFingerPrint(cacheKey *cachekey.CacheKey) (string, error) {
	cacheKeyJsonBytes, err := protojson.Marshal(cacheKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key with protojson: %w", err)
	}
	// This json unmarshal and marshal is to use encoding/json formatter to format the bytes[] returned by protojson
	// Do the json formatter because of https://developers.google.com/protocol-buffers/docs/reference/go/faq#unstable-json
	var v interface{}
	if err := json.Unmarshal(cacheKeyJsonBytes, &v); err != nil {
		return "", fmt.Errorf("failed to unmarshall cache key json bytes array: %w", err)
	}
	formattedCacheKeyBytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshall cache key with golang encoding/json : %w", err)
	}
	hash := sha256.New()
	hash.Write(formattedCacheKeyBytes)
	md := hash.Sum(nil)
	executionHashKey := hex.EncodeToString(md)
	return executionHashKey, nil
}

func GenerateCacheKey(
	inputs *pipelinespec.ExecutorInput_Inputs,
	outputs *pipelinespec.ExecutorInput_Outputs,
	outputParametersTypeMap map[string]string,
	cmdArgs []string, image string,
	pvcNames []string,
) (*cachekey.CacheKey, error) {
	cacheKey := cachekey.CacheKey{
		InputArtifactNames:   make(map[string]*cachekey.ArtifactNameList),
		InputParameterValues: make(map[string]*structpb.Value),
		OutputArtifactsSpec:  make(map[string]*pipelinespec.RuntimeArtifact),
		OutputParametersSpec: make(map[string]string),
	}

	for inputArtifactName, inputArtifactList := range inputs.GetArtifacts() {
		inputArtifactNameList := cachekey.ArtifactNameList{ArtifactNames: make([]string, 0)}
		for _, artifact := range inputArtifactList.Artifacts {
			inputArtifactNameList.ArtifactNames = append(inputArtifactNameList.ArtifactNames, artifact.GetName())
		}
		cacheKey.InputArtifactNames[inputArtifactName] = &inputArtifactNameList
	}

	for inputParameterName, inputParameterValue := range inputs.GetParameterValues() {
		cacheKey.InputParameterValues[inputParameterName] = inputParameterValue
	}

	for outputArtifactName, outputArtifactList := range outputs.GetArtifacts() {
		if len(outputArtifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		outputArtifact := outputArtifactList.Artifacts[0]
		outputArtifactWithUriWiped := pipelinespec.RuntimeArtifact{
			Name:     outputArtifact.GetName(),
			Type:     outputArtifact.GetType(),
			Metadata: outputArtifact.GetMetadata(),
		}
		cacheKey.OutputArtifactsSpec[outputArtifactName] = &outputArtifactWithUriWiped
	}

	for outputParameterName := range outputs.GetParameters() {
		outputParameterType, ok := outputParametersTypeMap[outputParameterName]
		if !ok {
			return nil, fmt.Errorf("unknown parameter %q found in ExecutorInput_Outputs", outputParameterName)
		}

		cacheKey.OutputParametersSpec[outputParameterName] = outputParameterType
	}

	cacheKey.ContainerSpec = &cachekey.ContainerSpec{
		Image:    image,
		CmdArgs:  cmdArgs,
		PvcNames: pvcNames,
	}

	return &cacheKey, nil
}
