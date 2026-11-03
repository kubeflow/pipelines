// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// This independent pre-promotion baseline is never regenerated from v2 or
// registered globally. Only contract and legacy-wire tests load it.
//
//go:embed testdata/legacy_descriptor.pb
var legacyDescriptor []byte

func legacyDescriptors(t *testing.T) *protoregistry.Files {
	t.Helper()
	require.Equal(t, "f5bc92c76ba552c2773dc58aed42fcd2c01abee1055067a2e805e0cb8bee852a", fmt.Sprintf("%x", sha256.Sum256(legacyDescriptor)))
	registry := new(protoregistry.Files)
	require.NoError(t, registerDescriptors(legacyDescriptor, registry))
	require.Equal(t, 10, registry.NumFiles())
	return registry
}

type descriptorResolver struct {
	local *protoregistry.Files
}

func (r descriptorResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if file, err := r.local.FindFileByPath(path); err == nil {
		return file, nil
	}
	return protoregistry.GlobalFiles.FindFileByPath(path)
}

func (r descriptorResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if descriptor, err := r.local.FindDescriptorByName(name); err == nil {
		return descriptor, nil
	}
	return protoregistry.GlobalFiles.FindDescriptorByName(name)
}

func registerDescriptors(data []byte, registry *protoregistry.Files) error {
	set := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(data, set); err != nil {
		return fmt.Errorf("decode frozen v2beta1 descriptors: %w; restore the checksum-pinned test snapshot", err)
	}
	pending := make(map[string]*descriptorpb.FileDescriptorProto, len(set.File))
	for _, file := range set.File {
		pending[file.GetName()] = file
	}
	visiting := make(map[string]bool)
	var register func(string) error
	register = func(name string) error {
		file, ok := pending[name]
		if !ok {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("dependency cycle in frozen v2beta1 descriptor %s; restore the checksum-pinned test snapshot", name)
		}
		visiting[name] = true
		for _, dependency := range file.Dependency {
			if err := register(dependency); err != nil {
				return err
			}
		}
		descriptor, err := protodesc.NewFile(file, descriptorResolver{local: registry})
		if err != nil {
			return fmt.Errorf("resolve frozen v2beta1 descriptor %s: %w; verify the snapshot checksum and load its external protobuf dependencies", name, err)
		}
		if err := registry.RegisterFile(descriptor); err != nil {
			return fmt.Errorf("register frozen v2beta1 descriptor %s: %w; remove duplicate legacy descriptor registration", name, err)
		}
		delete(pending, name)
		return nil
	}
	for _, file := range set.File {
		if err := register(file.GetName()); err != nil {
			return err
		}
	}
	return nil
}

func TestFrozenDescriptorsStayTestLocal(t *testing.T) {
	registry := legacyDescriptors(t)
	name := protoreflect.FullName(legacyRPCPackage + "Experiment")
	_, err := registry.FindDescriptorByName(name)
	require.NoError(t, err)
	_, err = protoregistry.GlobalFiles.FindDescriptorByName(name)
	require.ErrorIs(t, err, protoregistry.NotFound)
	_, err = protoregistry.GlobalTypes.FindMessageByName(name)
	require.ErrorIs(t, err, protoregistry.NotFound)
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
