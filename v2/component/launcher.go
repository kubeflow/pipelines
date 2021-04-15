// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package component contains types to run an ML component from the launcher
// binary.
package component

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/v2/metadata"
	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
	"google.golang.org/protobuf/encoding/protojson"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
)

// Launcher is used to launch KFP components. It handles the recording of the
// appropriate metadata for lineage.
type Launcher struct {
	options                 *LauncherOptions
	runtimeInfo             *runtimeInfo
	placeholderReplacements map[string]string
	metadataClient          *metadata.Client
	bucketConfig            *bucketConfig
}

// LauncherOptions are options used when creating Launcher.
type LauncherOptions struct {
	PipelineName      string
	PipelineRoot      string
	PipelineRunID     string
	PipelineTaskID    string
	TaskName          string
	ContainerImage    string
	MLMDServerAddress string
	MLMDServerPort    string
}

type bucketConfig struct {
	scheme     string
	bucketName string
	prefix     string
}

func (b *bucketConfig) bucketURL() string {
	u := b.scheme + b.bucketName

	if len(b.prefix) > 0 {
		u = fmt.Sprintf("%s?prefix=%s", u, b.prefix)
	}
	return u
}

func (b *bucketConfig) keyFromURI(uri string) (string, error) {
	prefixedBucket := b.scheme + path.Join(b.bucketName, b.prefix)
	if !strings.HasPrefix(uri, prefixedBucket) {
		return "", fmt.Errorf("URI %q does not have expected bucket prefix %q", uri, prefixedBucket)
	}

	key := strings.TrimLeft(strings.TrimPrefix(uri, prefixedBucket), "/")
	if len(key) == 0 {
		return "", fmt.Errorf("URI %q has empty key given prefixed bucket %q", uri, prefixedBucket)
	}
	return key, nil
}

func (b *bucketConfig) uriFromKey(blobKey string) string {
	return b.scheme + path.Join(b.bucketName, b.prefix, blobKey)
}

var bucketPattern = regexp.MustCompile(`^([a-z][a-z0-9]+:///?)([^/]+)(/[^ ?]*)?$`)

func parseBucketConfig(path string) (*bucketConfig, error) {
	ms := bucketPattern.FindStringSubmatch(path)
	if ms == nil || len(ms) != 4 {
		return nil, fmt.Errorf("Unrecognized pipeline root format: %q", path)
	}

	// TODO: Verify/add support for s3:// and file:///.
	if ms[1] != "gs://" {
		return nil, fmt.Errorf("Unsupported Cloud bucket: %q", path)
	}

	prefix := strings.TrimPrefix(ms[3], "/")
	if len(prefix) > 0 && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return &bucketConfig{
		scheme:     ms[1],
		bucketName: ms[2],
		prefix:     prefix,
	}, nil
}

func (o *LauncherOptions) validate() error {
	empty := func(s string) bool { return len(s) == 0 }
	err := func(s string) error { return fmt.Errorf("Must specify %s", s) }

	if empty(o.PipelineName) {
		return err("PipelineName")
	}
	if empty(o.PipelineRunID) {
		return err("PipelineRunID")
	}
	if empty(o.PipelineTaskID) {
		return err("PipelineTaskID")
	}
	if empty(o.PipelineRoot) {
		return err("PipelineRoot")
	}
	if empty(o.TaskName) {
		return err("TaskName")
	}
	if empty(o.MLMDServerAddress) {
		return err("MLMDServerAddress")
	}
	if empty(o.MLMDServerPort) {
		return err("MLMDServerPort")
	}
	return nil
}

const outputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"

// NewLauncher creates a new launcher object using the JSON-encoded runtimeInfo
// and specified options.
func NewLauncher(runtimeInfo string, options *LauncherOptions) (*Launcher, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}

	bc, err := parseBucketConfig(options.PipelineRoot)
	if err != nil {
		return nil, err
	}

	rt, err := parseRuntimeInfo(runtimeInfo)
	if err != nil {
		return nil, err
	}

	// Placeholder replacements.
	pr := make(map[string]string)

	metadataClient, err := metadata.NewClient(options.MLMDServerAddress, options.MLMDServerPort)
	if err != nil {
		return nil, err
	}

	return &Launcher{
		options:                 options,
		placeholderReplacements: pr,
		runtimeInfo:             rt,
		metadataClient:          metadataClient,
		bucketConfig:            bc,
	}, nil
}

// RunComponent runs the current KFP component using the specified command and
// arguments.
func (l *Launcher) RunComponent(ctx context.Context, cmd string, args ...string) error {
	executorInput, err := l.runtimeInfo.generateExecutorInput(l.generateOutputURI, outputMetadataFilepath)
	if err != nil {
		return fmt.Errorf("failure while generating ExecutorInput: %w", err)
	}

	if err := l.prepareInputs(ctx, executorInput); err != nil {
		return err
	}

	if err := l.prepareOutputs(ctx, executorInput); err != nil {
		return err
	}

	// Update command.
	for placeholder, replacement := range l.placeholderReplacements {
		cmd = strings.ReplaceAll(cmd, placeholder, replacement)
	}

	// Update args.
	for i := range args {
		arg := args[i]
		for placeholder, replacement := range l.placeholderReplacements {
			arg = strings.ReplaceAll(arg, placeholder, replacement)
		}
		args[i] = arg
	}

	// Record Execution in MLMD.
	// TODO(neuromage): Refactor launcher.go and split these functions up into
	// testable units.
	pipeline, err := l.metadataClient.GetPipeline(ctx, l.options.PipelineName, l.options.PipelineRunID)
	if err != nil {
		return err
	}

	ecfg := &metadata.ExecutionConfig{
		InputParameters: &metadata.Parameters{
			IntParameters:    make(map[string]int64),
			StringParameters: make(map[string]string),
			DoubleParameters: make(map[string]float64),
		},
	}

	for _, artifactList := range executorInput.Inputs.Artifacts {
		for _, artifact := range artifactList.Artifacts {
			id, err := strconv.ParseInt(artifact.Name, 10, 64)
			if err != nil {
				return fmt.Errorf("unable to parse input artifact id from %q: %w", id, err)
			}
			ecfg.InputArtifactIDs = append(ecfg.InputArtifactIDs, id)
		}
	}

	for name, parameter := range executorInput.Inputs.Parameters {
		switch t := parameter.Value.(type) {
		case *pipeline_spec.Value_StringValue:
			ecfg.InputParameters.StringParameters[name] = parameter.GetStringValue()
		case *pipeline_spec.Value_IntValue:
			ecfg.InputParameters.IntParameters[name] = parameter.GetIntValue()
		case *pipeline_spec.Value_DoubleValue:
			ecfg.InputParameters.DoubleParameters[name] = parameter.GetDoubleValue()
		default:
			return fmt.Errorf("unknown parameter type: %T", t)
		}
	}

	execution, err := l.metadataClient.CreateExecution(ctx, pipeline, l.options.TaskName, l.options.PipelineTaskID, l.options.ContainerImage, ecfg)
	if err != nil {
		return err
	}

	executor := exec.Command(cmd, args...)

	executor.Stdin = os.Stdin
	executor.Stdout = os.Stdout
	executor.Stderr = os.Stderr
	defer glog.Flush()
	if err := executor.Run(); err != nil {
		return err
	}

	executorOutput, err := getExecutorOutput()
	if err != nil {
		return err
	}

	for name, parameter := range executorOutput.Parameters {
		var value string
		switch t := parameter.Value.(type) {
		case *pipeline_spec.Value_StringValue:
			value = parameter.GetStringValue()
		case *pipeline_spec.Value_DoubleValue:
			value = strconv.FormatFloat(parameter.GetDoubleValue(), 'f', -1, 64)
		case *pipeline_spec.Value_IntValue:
			value = strconv.FormatInt(parameter.GetIntValue(), 10)
		default:
			return fmt.Errorf("unknown PipelineSpec Value type %T", t)
		}

		outputParam, ok := l.runtimeInfo.OutputParameters[name]
		if !ok {
			return fmt.Errorf("unknown parameter %q found in ExecutorOutput", name)
		}
		filename := outputParam.Path
		if err := ioutil.WriteFile(filename, []byte(value), 0644); err != nil {
			return fmt.Errorf("failed to write output parameter %q to file %q: %w", name, filename, err)
		}
	}

	bucket, err := blob.OpenBucket(context.Background(), l.bucketConfig.bucketURL())
	if err != nil {
		return fmt.Errorf("Failed to open bucket %q: %v", l.bucketConfig.bucketName, err)
	}
	defer bucket.Close()

	// Register artifacts with MLMD.
	outputArtifacts := make([]*metadata.OutputArtifact, 0, len(l.runtimeInfo.OutputArtifacts))
	for name, artifactList := range executorInput.Outputs.Artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		// TODO: Support multiple artifacts someday, probably through the v2 engine.
		outputArtifact := artifactList.Artifacts[0]

		if list, ok := executorOutput.Artifacts[name]; ok && len(list.Artifacts) > 0 {
			mergeRuntimeArtifacts(list.Artifacts[0], outputArtifact)
		}

		localDir, err := localPathForURI(outputArtifact.Uri)
		if err != nil {
			glog.Warningf("Output Artifact %q does not have a recognized storage URI %q. Skipping uploading to remote storage.", name, outputArtifact.Uri)
			continue
		}

		blobKey, err := l.bucketConfig.keyFromURI(outputArtifact.Uri)
		if err := uploadBlob(ctx, bucket, localDir, blobKey); err != nil {
			return fmt.Errorf("failed to upload output artifact %q to remote storage URI %q: %w", name, outputArtifact.Uri, err)
		}

		// Write out the metadata.
		metadataErr := func(err error) error {
			return fmt.Errorf("unable to produce MLMD artifact for output %q: %w", name, err)
		}
		mlmdArtifact, err := toMLMDArtifact(outputArtifact)
		if err != nil {
			return metadataErr(err)
		}

		// TODO(neuromage): Consider batching these instead of recording one by one.
		schema, err := getRuntimeArtifactSchema(outputArtifact)
		if err != nil {
			return fmt.Errorf("failed to determine schema for output %q: %w", name, err)
		}
		mlmdArtifact, err = l.metadataClient.RecordArtifact(ctx, schema, mlmdArtifact)
		if err != nil {
			return metadataErr(err)
		}
		outputArtifacts = append(outputArtifacts, &metadata.OutputArtifact{Artifact: mlmdArtifact, Schema: outputArtifact.Type.GetInstanceSchema()})

		rtoa, ok := l.runtimeInfo.OutputArtifacts[name]
		if !ok {
			return metadataErr(errors.New("unable to find output artifact in RuntimeInfo"))
		}
		if err := os.MkdirAll(path.Dir(rtoa.MetadataPath), 0644); err != nil {
			return metadataErr(err)
		}

		b, err := protojson.Marshal(mlmdArtifact)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(rtoa.MetadataPath, b, 0644); err != nil {
			return err
		}
	}

	// Read output parameters.
	outputParameters := &metadata.Parameters{
		IntParameters:    make(map[string]int64),
		StringParameters: make(map[string]string),
		DoubleParameters: make(map[string]float64),
	}

	for n, op := range l.runtimeInfo.OutputParameters {
		b, err := ioutil.ReadFile(op.Path)
		if err != nil {
			return err
		}
		switch op.Type {
		case "STRING":
			outputParameters.StringParameters[n] = string(b)
		case "INT":
			i, err := strconv.ParseInt(string(b), 10, 0)
			if err != nil {
				return err
			}
			outputParameters.IntParameters[n] = i
		case "DOUBLE":
			f, err := strconv.ParseFloat(string(b), 0)
			if err != nil {
				return err
			}
			outputParameters.DoubleParameters[n] = f
		}

	}

	return l.metadataClient.PublishExecution(ctx, execution, outputParameters, outputArtifacts)
}

func (l *Launcher) generateOutputURI(name string) string {
	blobKey := path.Join(l.options.PipelineName, l.options.PipelineRunID, l.options.TaskName, name)
	return l.bucketConfig.uriFromKey(blobKey)
}

func localPathForURI(uri string) (string, error) {
	if strings.HasPrefix(uri, "gs://") {
		return "/gcs/" + strings.TrimPrefix(uri, "gs://"), nil
	}
	// TODO(capri-xiyue): Re-enable when support lands.
	// if strings.HasPrefix(uri, "minio://") {
	// 	return "/minio/" + strings.TrimPrefix(uri, "minio://"), nil
	// }
	// if strings.HasPrefix(uri, "s3://") {
	// 	return "/s3/" + strings.TrimPrefix(uri, "s3://"), nil
	// }
	return "", fmt.Errorf("found URI with unsupported storage scheme: %s", uri)
}

func (l *Launcher) prepareInputs(ctx context.Context, executorInput *pipeline_spec.ExecutorInput) error {
	executorInputJSON, err := protojson.Marshal(executorInput)
	if err != nil {
		return fmt.Errorf("failed to convert ExecutorInput into JSON: %w", err)
	}
	l.placeholderReplacements["{{$}}"] = string(executorInputJSON)

	bucket, err := blob.OpenBucket(context.Background(), l.bucketConfig.bucketURL())
	if err != nil {
		return fmt.Errorf("Failed to open bucket %q: %v", l.bucketConfig.bucketName, err)
	}
	defer bucket.Close()

	// Read input artifact metadata.
	for name, artifactList := range executorInput.Inputs.Artifacts {
		// TODO(neuromage): Support concat-based placholders for arguments.
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		inputArtifact := artifactList.Artifacts[0]

		// Prepare input uri placeholder.
		key := fmt.Sprintf(`{{$.inputs.artifacts['%s'].uri}}`, name)
		l.placeholderReplacements[key] = inputArtifact.Uri

		localPath, err := localPathForURI(inputArtifact.Uri)
		if err != nil {
			glog.Warningf("Input Artifact %q does not have a recognized storage URI %q. Skipping downloading to local path.", name, inputArtifact.Uri)
			continue
		}

		// Prepare input path placeholder.
		key = fmt.Sprintf(`{{$.inputs.artifacts['%s'].path}}`, name)
		l.placeholderReplacements[key] = localPath

		// Copy artifact to local storage.
		copyErr := func(err error) error {
			return fmt.Errorf("failed to download input artifact %q from remote storage URI %q: %w", name, inputArtifact.Uri, err)
		}
		// TODO: Selectively copy artifacts for which .path was actually specified
		// on the command line.
		blobKey, err := l.bucketConfig.keyFromURI(inputArtifact.Uri)
		if err != nil {
			return copyErr(err)
		}

		if err := downloadBlob(ctx, bucket, localPath, blobKey); err != nil {
			return copyErr(err)
		}
	}

	// Prepare input parameter placeholders.
	for name, parameter := range executorInput.Inputs.Parameters {
		key := fmt.Sprintf(`{{$.inputs.parameters['%s']}}`, name)
		switch t := parameter.Value.(type) {
		case *pipeline_spec.Value_StringValue:
			l.placeholderReplacements[key] = parameter.GetStringValue()
		case *pipeline_spec.Value_DoubleValue:
			l.placeholderReplacements[key] = strconv.FormatFloat(parameter.GetDoubleValue(), 'f', -1, 64)
		case *pipeline_spec.Value_IntValue:
			l.placeholderReplacements[key] = strconv.FormatInt(parameter.GetIntValue(), 10)
		default:
			return fmt.Errorf("unknown PipelineSpec Value type %T", t)
		}
	}

	return nil
}

func (l *Launcher) prepareOutputs(ctx context.Context, executorInput *pipeline_spec.ExecutorInput) error {
	for name, parameter := range executorInput.Outputs.Parameters {
		key := fmt.Sprintf(`{{$.outputs.parameters['%s'].output_file}}`, name)
		l.placeholderReplacements[key] = parameter.OutputFile

		dir := filepath.Dir(parameter.OutputFile)
		if err := os.MkdirAll(dir, 0644); err != nil {
			return fmt.Errorf("failed to create directory %q for output parameter %q: %w", dir, name, err)
		}
	}

	for name, artifactList := range executorInput.Outputs.Artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		outputArtifact := artifactList.Artifacts[0]

		key := fmt.Sprintf(`{{$.outputs.artifacts['%s'].uri}}`, name)
		l.placeholderReplacements[key] = outputArtifact.Uri

		localPath, err := localPathForURI(outputArtifact.Uri)
		if err != nil {
			return fmt.Errorf("failed to generate local storage path for output artifact %q with URI %q: %w", name, outputArtifact.Uri, err)
		}

		if err := os.MkdirAll(filepath.Base(localPath), 0644); err != nil {
			return fmt.Errorf("unable to create directory %q for output artifact %q: %w", localPath, name, err)
		}

		key = fmt.Sprintf(`{{$.outputs.artifacts['%s'].path}}`, name)
		l.placeholderReplacements[key] = localPath
	}

	return nil
}

func getRuntimeArtifactSchema(rta *pipeline_spec.RuntimeArtifact) (string, error) {
	switch t := rta.Type.Kind.(type) {
	case *pipeline_spec.ArtifactTypeSchema_InstanceSchema:
		return t.InstanceSchema, nil
	case *pipeline_spec.ArtifactTypeSchema_SchemaTitle:
		return "title: " + t.SchemaTitle, nil
	case *pipeline_spec.ArtifactTypeSchema_SchemaUri:
		return "", fmt.Errorf("SchemaUri is unsupported, found in RuntimeArtifact %+v", rta)
	default:
		return "", fmt.Errorf("unknown type %T in RuntimeArtifact %+v", t, rta)
	}
}

func mergeRuntimeArtifacts(src, dst *pipeline_spec.RuntimeArtifact) {
	if len(src.Uri) > 0 {
		dst.Uri = src.Uri
	}

	if src.Metadata != nil {
		if dst.Metadata == nil {
			dst.Metadata = src.Metadata
		} else {
			for k, v := range src.Metadata.Fields {
				dst.Metadata.Fields[k] = v
			}
		}
	}
}

// TODO(neuromage): Move these helper functions to a storage package and add tests.
func uploadFile(ctx context.Context, bucket *blob.Bucket, localFilePath, blobFilePath string) error {
	errorF := func(err error) error {
		return fmt.Errorf("uploadFile(): unable to complete copying %q to remote storage %q: %w", localFilePath, blobFilePath, err)
	}

	w, err := bucket.NewWriter(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open writer for bucket: %w", err))
	}

	r, err := os.Open(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for reading: %w", localFilePath, err))
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	if err = w.Close(); err != nil {
		return errorF(fmt.Errorf("failed to close Writer for bucket: %w", err))
	}

	return nil
}

func downloadFile(ctx context.Context, bucket *blob.Bucket, blobFilePath, localFilePath string) error {
	errorF := func(err error) error {
		return fmt.Errorf("downloadFile(): unable to complete copying %q to local storage %q: %w", blobFilePath, localFilePath, err)
	}

	r, err := bucket.NewReader(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open reader for bucket: %w", err))
	}
	defer r.Close()

	localDir := filepath.Dir(localFilePath)
	if err := os.MkdirAll(localDir, 0644); err != nil {
		return errorF(fmt.Errorf("failed to create local directory %q: %w", localDir, err))
	}

	w, err := os.Create(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for writing: %w", localFilePath, err))
	}
	defer w.Close()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	return nil
}

func uploadBlob(ctx context.Context, bucket *blob.Bucket, localPath, blobPath string) error {
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("unable to stat local filepath %q: %w", localPath, err)
	}

	if !fileInfo.IsDir() {
		return uploadFile(ctx, bucket, localPath, blobPath)
	}

	// localPath is a directory.
	files, err := ioutil.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("unable to list local directory %q: %w", localPath, err)
	}

	for _, f := range files {
		if f.IsDir() {
			// TODO
			continue
		}

		blobFilePath := filepath.Join(blobPath, filepath.Base(f.Name()))
		localFilePath := filepath.Join(localPath, f.Name())
		if err := uploadFile(ctx, bucket, localFilePath, blobFilePath); err != nil {
			return err
		}
	}

	return nil
}

func downloadBlob(ctx context.Context, bucket *blob.Bucket, localDir, blobDir string) error {
	iter := bucket.List(&blob.ListOptions{Prefix: blobDir})

	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to list objects in remote storage %q: %w", blobDir, err)
		}

		if obj.IsDir {
			continue
		}

		var localFilePath string
		if obj.Key == blobDir {
			// Artifact URI is a file on Remote Storage.
			localFilePath = localDir
		} else {
			// Artifact URI is a directory on Remote Storage.
			localFilePath = filepath.Join(localDir, path.Base(obj.Key))
		}
		if err := downloadFile(ctx, bucket, obj.Key, localFilePath); err != nil {
			return err
		}
	}

	return nil
}

func getExecutorOutput() (*pipeline_spec.ExecutorOutput, error) {
	executorOutput := &pipeline_spec.ExecutorOutput{
		Parameters: map[string]*pipeline_spec.Value{},
		Artifacts:  map[string]*pipeline_spec.ArtifactList{},
	}

	_, err := os.Stat(outputMetadataFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return an empty ExecutorOutput.
			return executorOutput, nil
		} else {
			return nil, fmt.Errorf("failed to stat output metadata file %q: %w", outputMetadataFilepath, err)
		}
	}

	b, err := ioutil.ReadFile(outputMetadataFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output metadata file %q: %w", outputMetadataFilepath, err)
	}

	if err := protojson.Unmarshal(b, executorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshall ExecutorOutput in file %q: %w", outputMetadataFilepath, err)
	}

	return executorOutput, nil
}
