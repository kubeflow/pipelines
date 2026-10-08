// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

// Package v2beta1 registers the frozen legacy contracts for gRPC reflection.
// All executable client and server behavior belongs to v2.
package v2beta1

import (
	_ "embed"
	"fmt"

	// Load the canonical API's external protobuf dependencies before resolving
	// the frozen descriptors. No legacy generated message code is needed.
	_ "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Frozen legacy schema, pinned by the SHA-256 assertion in descriptors_test.go.
// This snapshot must not be regenerated from v2: it is the independent
// compatibility baseline for already-compiled legacy clients.
//
//go:embed legacy_descriptor.pb
var legacyDescriptor []byte

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
		return fmt.Errorf("decode frozen v2beta1 descriptors: %w; restore legacy_descriptor.pb matching the SHA-256 in descriptors_test.go", err)
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
			return fmt.Errorf("dependency cycle in frozen v2beta1 descriptor %s; restore the checksum-pinned legacy_descriptor.pb", name)
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

// Dynamic types preserve legacy Any type URLs without keeping a second set of
// generated Go messages. Normal API requests still decode directly into v2.
func registerTypes(registry *protoregistry.Types, messages protoreflect.MessageDescriptors, enums protoreflect.EnumDescriptors) error {
	for i := 0; i < enums.Len(); i++ {
		if err := registry.RegisterEnum(dynamicpb.NewEnumType(enums.Get(i))); err != nil {
			return err
		}
	}
	for i := 0; i < messages.Len(); i++ {
		message := messages.Get(i)
		if err := registry.RegisterMessage(dynamicpb.NewMessageType(message)); err != nil {
			return err
		}
		if err := registerTypes(registry, message.Messages(), message.Enums()); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	if err := registerDescriptors(legacyDescriptor, protoregistry.GlobalFiles); err != nil {
		panic(err)
	}
	var files []protoreflect.FileDescriptor
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if file.Package() == "kubeflow.pipelines.backend.api.v2beta1" {
			files = append(files, file)
		}
		return true
	})
	// GlobalFiles and GlobalTypes share a lock; register outside RangeFiles.
	for _, file := range files {
		if err := registerTypes(protoregistry.GlobalTypes, file.Messages(), file.Enums()); err != nil {
			panic(err)
		}
	}
}
