package component

import "github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"

type cacheKey struct {
	inputArtifactNames map[string]artifactNameList
	inputParameters map[string]pipeline_spec.Value
	outputArtifactsSpec map[string]pipeline_spec.RuntimeArtifact
	outputParametersSpec map[string]string
	containerSpec containerSpec
}

type artifactNameList struct {
	// A list of artifact Names.
	artifactNames []string
}

type containerSpec struct {
	image   string
	commands string
	args []string
	// TO DO , remove cpuLimit and memoryLimit once SDK adds such info into env variables
	cpuLimit float64
	memoryLimit float64
	env      []envVar
}

type envVar struct {
	name string
	value string
}