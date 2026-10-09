// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package v2beta1

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestFrozenDescriptorSnapshot(t *testing.T) {
	// Keep the historical compatibility baseline independent of v2 generation.
	require.Equal(t, "f5bc92c76ba552c2773dc58aed42fcd2c01abee1055067a2e805e0cb8bee852a", fmt.Sprintf("%x", sha256.Sum256(legacyDescriptor)))
	registry := new(protoregistry.Files)
	require.NoError(t, registerDescriptors(legacyDescriptor, registry))
	require.Equal(t, 10, registry.NumFiles())
	for _, service := range []string{"AuthService", "ArtifactService", "ExperimentService", "PipelineService", "RecurringRunService", "RunService", "ReportService"} {
		descriptor, err := registry.FindDescriptorByName(protoreflect.FullName("kubeflow.pipelines.backend.api.v2beta1." + service))
		require.NoError(t, err)
		require.Equal(t, protoreflect.FullName("kubeflow.pipelines.backend.api.v2beta1"), descriptor.ParentFile().Package())
	}
	_, err := protoregistry.GlobalTypes.FindMessageByName("kubeflow.pipelines.backend.api.v2beta1.Predicate.IntValues")
	require.NoError(t, err)
	_, err = protoregistry.GlobalTypes.FindEnumByName("kubeflow.pipelines.backend.api.v2beta1.Experiment.StorageState")
	require.NoError(t, err)
	_, err = registry.FindFileByPath("backend/api/v2/run.proto")
	require.ErrorIs(t, err, protoregistry.NotFound)
}

func TestLegacyAnyJSONRoundTrip(t *testing.T) {
	input := `{"@type":"type.googleapis.com/kubeflow.pipelines.backend.api.v2beta1.Experiment","display_name":"legacy"}`
	message := new(anypb.Any)
	require.NoError(t, protojson.Unmarshal([]byte(input), message))
	decoded, err := anypb.UnmarshalNew(message, proto.UnmarshalOptions{})
	require.NoError(t, err)
	require.Equal(t, "kubeflow.pipelines.backend.api.v2beta1.Experiment", string(decoded.ProtoReflect().Descriptor().FullName()))
	output, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(message)
	require.NoError(t, err)
	require.JSONEq(t, input, string(output))
}

func TestRegisterDescriptorsRejectsInvalidData(t *testing.T) {
	require.ErrorContains(t, registerDescriptors([]byte{0xff}, new(protoregistry.Files)), "decode frozen v2beta1 descriptors")
	cycle := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		{Name: proto.String("first.proto"), Dependency: []string{"second.proto"}},
		{Name: proto.String("second.proto"), Dependency: []string{"first.proto"}},
	}}
	data, err := proto.Marshal(cycle)
	require.NoError(t, err)
	require.ErrorContains(t, registerDescriptors(data, new(protoregistry.Files)), "dependency cycle")
}
