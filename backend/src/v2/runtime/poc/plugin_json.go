package poc

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type taskExecutionFields TaskExecutionRequest

type taskExecutionJSON struct {
	*taskExecutionFields
	Container     json.RawMessage `json:"container,omitempty"`
	ExecutorInput json.RawMessage `json:"executorInput,omitempty"`
}

// MarshalJSON preserves protobuf oneofs within the durable task request.
func (request TaskExecutionRequest) MarshalJSON() ([]byte, error) {
	fields := taskExecutionFields(request)
	wire := taskExecutionJSON{taskExecutionFields: &fields}
	var err error
	wire.Container, err = executionProtoJSON(request.Container)
	if err != nil {
		return nil, err
	}
	wire.ExecutorInput, err = executionProtoJSON(request.ExecutorInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire)
}

// UnmarshalJSON restores protobuf messages without exposing framework objects.
func (request *TaskExecutionRequest) UnmarshalJSON(raw []byte) error {
	fields := taskExecutionFields{}
	wire := taskExecutionJSON{taskExecutionFields: &fields}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	if hasExecutionProtoJSON(wire.Container) {
		fields.Container = &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
		if err := protojson.Unmarshal(wire.Container, fields.Container); err != nil {
			return fmt.Errorf("cannot decode resolved container: %w", err)
		}
	}
	input, err := executionInputFromJSON(wire.ExecutorInput)
	if err != nil {
		return err
	}
	fields.ExecutorInput = input
	*request = TaskExecutionRequest(fields)
	return nil
}

type dockerExecutionFields dockerExecutionData

type dockerExecutionJSON struct {
	*dockerExecutionFields
	ExecutorInput json.RawMessage `json:"executorInput,omitempty"`
}

func (data dockerExecutionData) MarshalJSON() ([]byte, error) {
	fields := dockerExecutionFields(data)
	input, err := executionProtoJSON(data.ExecutorInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dockerExecutionJSON{dockerExecutionFields: &fields, ExecutorInput: input})
}

func (data *dockerExecutionData) UnmarshalJSON(raw []byte) error {
	fields := dockerExecutionFields{}
	wire := dockerExecutionJSON{dockerExecutionFields: &fields}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	input, err := executionInputFromJSON(wire.ExecutorInput)
	if err != nil {
		return err
	}
	fields.ExecutorInput = input
	*data = dockerExecutionData(fields)
	return nil
}

func executionProtoJSON(message proto.Message) (json.RawMessage, error) {
	if message == nil || !message.ProtoReflect().IsValid() {
		return nil, nil
	}
	return protojson.Marshal(message)
}

func hasExecutionProtoJSON(raw json.RawMessage) bool {
	return len(raw) != 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func executionInputFromJSON(raw json.RawMessage) (*pipelinespec.ExecutorInput, error) {
	if !hasExecutionProtoJSON(raw) {
		return nil, nil
	}
	input := &pipelinespec.ExecutorInput{}
	if err := protojson.Unmarshal(raw, input); err != nil {
		return nil, fmt.Errorf("cannot decode resolved executor input: %w", err)
	}
	return input, nil
}
