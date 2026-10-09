// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func canonicalDescriptorName(name protoreflect.FullName) protoreflect.FullName {
	if string(name) == strings.TrimSuffix(legacyRPCPackage, ".") {
		return protoreflect.FullName(strings.TrimSuffix(canonicalRPCPackage, "."))
	}
	if strings.HasPrefix(string(name), legacyRPCPackage) {
		return protoreflect.FullName(canonicalRPCPackage + strings.TrimPrefix(string(name), legacyRPCPackage))
	}
	return name
}

func legacyEnumsCompatible(old, current protoreflect.EnumDescriptors) error {
	for i := 0; i < old.Len(); i++ {
		previous := old.Get(i)
		next := current.ByName(previous.Name())
		if next == nil {
			return fmt.Errorf("legacy enum %s was removed; retain it or add an adapter", previous.FullName())
		}
		for j := 0; j < previous.Values().Len(); j++ {
			value := previous.Values().Get(j)
			candidate := next.Values().ByName(value.Name())
			if candidate == nil || candidate.Number() != value.Number() {
				return fmt.Errorf("legacy enum value %s changed; retain its name and number", value.FullName())
			}
		}
	}
	return nil
}

func legacyMessagesCompatible(old, current protoreflect.MessageDescriptors) error {
	for i := 0; i < old.Len(); i++ {
		previous := old.Get(i)
		next := current.ByName(previous.Name())
		if next == nil || previous.IsMapEntry() != next.IsMapEntry() {
			return fmt.Errorf("legacy message %s changed or was removed; retain it or add an adapter", previous.FullName())
		}
		for j := 0; j < previous.Fields().Len(); j++ {
			field := previous.Fields().Get(j)
			candidate := next.Fields().ByNumber(field.Number())
			if candidate == nil || candidate.Name() != field.Name() || candidate.JSONName() != field.JSONName() ||
				candidate.Kind() != field.Kind() || candidate.Cardinality() != field.Cardinality() ||
				candidate.HasPresence() != field.HasPresence() || candidate.IsMap() != field.IsMap() ||
				candidate.IsPacked() != field.IsPacked() || fmt.Sprint(candidate.Default()) != fmt.Sprint(field.Default()) {
				return fmt.Errorf("legacy field %s changed; retain its number, type, cardinality, presence, JSON name and default", field.FullName())
			}
			oldOneof, newOneof := field.ContainingOneof(), candidate.ContainingOneof()
			if (oldOneof == nil) != (newOneof == nil) || (oldOneof != nil && (oldOneof.Name() != newOneof.Name() || oldOneof.IsSynthetic() != newOneof.IsSynthetic())) {
				return fmt.Errorf("legacy oneof for %s changed; retain the existing oneof membership", field.FullName())
			}
			if field.Message() != nil && canonicalDescriptorName(field.Message().FullName()) != candidate.Message().FullName() {
				return fmt.Errorf("legacy message type for %s changed; retain the referenced message", field.FullName())
			}
			if field.Enum() != nil {
				oldDefault, newDefault := field.DefaultEnumValue(), candidate.DefaultEnumValue()
				if canonicalDescriptorName(field.Enum().FullName()) != candidate.Enum().FullName() ||
					(oldDefault == nil) != (newDefault == nil) ||
					(oldDefault != nil && oldDefault.Name() != newDefault.Name()) {
					return fmt.Errorf("legacy enum type for %s changed; retain the referenced enum and default", field.FullName())
				}
			}
		}
		for j := 0; j < next.Fields().Len(); j++ {
			field := next.Fields().Get(j)
			if previous.Fields().ByNumber(field.Number()) == nil && field.Cardinality() == protoreflect.Required {
				return fmt.Errorf("new required field %s breaks old clients; make it optional", field.FullName())
			}
		}
		if err := legacyMessagesCompatible(previous.Messages(), next.Messages()); err != nil {
			return err
		}
		if err := legacyEnumsCompatible(previous.Enums(), next.Enums()); err != nil {
			return err
		}
	}
	return nil
}

func normalizedHTTPBindings(method protoreflect.MethodDescriptor) []*annotations.HttpRule {
	if !proto.HasExtension(method.Options(), annotations.E_Http) {
		return nil
	}
	var bindings []*annotations.HttpRule
	var collect func(*annotations.HttpRule)
	path := func(value string) string {
		if value == legacyAPIPath || strings.HasPrefix(value, legacyAPIPath+"/") {
			return canonicalAPIPath + strings.TrimPrefix(value, legacyAPIPath)
		}
		return value
	}
	collect = func(rule *annotations.HttpRule) {
		binding := proto.Clone(rule).(*annotations.HttpRule)
		binding.AdditionalBindings = nil
		binding.Selector = string(canonicalDescriptorName(protoreflect.FullName(binding.Selector)))
		switch pattern := binding.Pattern.(type) {
		case *annotations.HttpRule_Get:
			pattern.Get = path(pattern.Get)
		case *annotations.HttpRule_Post:
			pattern.Post = path(pattern.Post)
		case *annotations.HttpRule_Put:
			pattern.Put = path(pattern.Put)
		case *annotations.HttpRule_Patch:
			pattern.Patch = path(pattern.Patch)
		case *annotations.HttpRule_Delete:
			pattern.Delete = path(pattern.Delete)
		case *annotations.HttpRule_Custom:
			if pattern.Custom != nil {
				pattern.Custom.Path = path(pattern.Custom.Path)
			}
		}
		bindings = append(bindings, binding)
		for _, additional := range rule.AdditionalBindings {
			collect(additional)
		}
	}
	collect(proto.GetExtension(method.Options(), annotations.E_Http).(*annotations.HttpRule))
	return bindings
}

// A frozen contract is a lower bound, not a ban on additive v2 development.
func legacyFileCompatible(old, current protoreflect.FileDescriptor) error {
	if canonicalDescriptorName(old.Package()) != current.Package() || old.Syntax() != current.Syntax() {
		return fmt.Errorf("legacy package or syntax changed; retain the contract or add an adapter")
	}
	if err := legacyMessagesCompatible(old.Messages(), current.Messages()); err != nil {
		return err
	}
	if err := legacyEnumsCompatible(old.Enums(), current.Enums()); err != nil {
		return err
	}
	for i := 0; i < old.Services().Len(); i++ {
		service := old.Services().Get(i)
		next := current.Services().ByName(service.Name())
		if next == nil {
			return fmt.Errorf("legacy service %s was removed; retain its registration", service.FullName())
		}
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			candidate := next.Methods().ByName(method.Name())
			if candidate == nil || canonicalDescriptorName(method.Input().FullName()) != candidate.Input().FullName() ||
				canonicalDescriptorName(method.Output().FullName()) != candidate.Output().FullName() ||
				method.IsStreamingClient() != candidate.IsStreamingClient() || method.IsStreamingServer() != candidate.IsStreamingServer() {
				return fmt.Errorf("legacy method %s changed; retain its input, output and streaming contract", method.FullName())
			}
			for _, binding := range normalizedHTTPBindings(method) {
				found := false
				for _, currentBinding := range normalizedHTTPBindings(candidate) {
					found = found || proto.Equal(binding, currentBinding)
				}
				if !found {
					return fmt.Errorf("legacy HTTP binding for %s changed; retain its route and body mapping", method.FullName())
				}
			}
		}
	}
	return nil
}

func TestLegacyAPIContractCompatibility(t *testing.T) {
	files := 0
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(file.Path(), "backend/api/v2beta1/") {
			return true
		}
		files++
		t.Run(file.Path(), func(t *testing.T) {
			canonical, err := protoregistry.GlobalFiles.FindFileByPath(strings.Replace(file.Path(), "/v2beta1/", "/v2/", 1))
			require.NoError(t, err)
			require.NoError(t, legacyFileCompatible(file, canonical))
		})
		return true
	})
	require.Equal(t, 10, files)
}

func contractFixture(t *testing.T, pkg string) *descriptorpb.FileDescriptorProto {
	t.Helper()
	options := new(descriptorpb.MethodOptions)
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/apis/" + pkg + "/runs"}, Body: "*"})
	return &descriptorpb.FileDescriptorProto{
		Name: proto.String(pkg + ".proto"), Package: proto.String("kubeflow.pipelines.backend.api." + pkg), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Run"), Field: []*descriptorpb.FieldDescriptorProto{{
			Name: proto.String("run_id"), JsonName: proto.String("runId"), Number: proto.Int32(1),
			Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		}}}},
		EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("State"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("UNKNOWN"), Number: proto.Int32(0)}}}},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("RunService"), Method: []*descriptorpb.MethodDescriptorProto{{
			Name: proto.String("Create"), InputType: proto.String(".kubeflow.pipelines.backend.api." + pkg + ".Run"), OutputType: proto.String(".kubeflow.pipelines.backend.api." + pkg + ".Run"), Options: options,
		}}}},
	}
}

func TestLegacyContractAllowsAdditionsButRejectsBreakingChanges(t *testing.T) {
	legacy, err := protodesc.NewFile(contractFixture(t, "v2beta1"), nil)
	require.NoError(t, err)
	for _, tc := range []struct {
		name      string
		change    func(*descriptorpb.FileDescriptorProto)
		wantError string
	}{
		{"optional field", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType[0].Field = append(f.MessageType[0].Field, &descriptorpb.FieldDescriptorProto{Name: proto.String("extra"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()})
		}, ""},
		{"enum value", func(f *descriptorpb.FileDescriptorProto) {
			f.EnumType[0].Value = append(f.EnumType[0].Value, &descriptorpb.EnumValueDescriptorProto{Name: proto.String("NEW"), Number: proto.Int32(1)})
		}, ""},
		{"RPC", func(f *descriptorpb.FileDescriptorProto) {
			method := proto.Clone(f.Service[0].Method[0]).(*descriptorpb.MethodDescriptorProto)
			method.Name = proto.String("Get")
			f.Service[0].Method = append(f.Service[0].Method, method)
		}, ""},
		{"message", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType = append(f.MessageType, &descriptorpb.DescriptorProto{Name: proto.String("Extra")})
		}, ""},
		{"extra HTTP binding", func(f *descriptorpb.FileDescriptorProto) {
			rule := proto.GetExtension(f.Service[0].Method[0].Options, annotations.E_Http).(*annotations.HttpRule)
			rule.AdditionalBindings = append(rule.AdditionalBindings, &annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/apis/v2/extra"}})
		}, ""},
		{"field removed", func(f *descriptorpb.FileDescriptorProto) { f.MessageType[0].Field = nil }, "legacy field"},
		{"field renumbered", func(f *descriptorpb.FileDescriptorProto) { f.MessageType[0].Field[0].Number = proto.Int32(2) }, "legacy field"},
		{"field type", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType[0].Field[0].Type = descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()
		}, "legacy field"},
		{"JSON name", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType[0].Field[0].JsonName = proto.String("different")
		}, "legacy field"},
		{"cardinality", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType[0].Field[0].Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
		}, "legacy field"},
		{"enum removal", func(f *descriptorpb.FileDescriptorProto) { f.EnumType = nil }, "legacy enum"},
		{"enum value rename", func(f *descriptorpb.FileDescriptorProto) { f.EnumType[0].Value[0].Name = proto.String("RENAMED") }, "legacy enum value"},
		{"service removal", func(f *descriptorpb.FileDescriptorProto) { f.Service = nil }, "legacy service"},
		{"method removal", func(f *descriptorpb.FileDescriptorProto) { f.Service[0].Method = nil }, "legacy method"},
		{"streaming", func(f *descriptorpb.FileDescriptorProto) { f.Service[0].Method[0].ServerStreaming = proto.Bool(true) }, "legacy method"},
		{"response type", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType = append(f.MessageType, &descriptorpb.DescriptorProto{Name: proto.String("Other")})
			f.Service[0].Method[0].OutputType = proto.String(".kubeflow.pipelines.backend.api.v2.Other")
		}, "legacy method"},
		{"HTTP route", func(f *descriptorpb.FileDescriptorProto) {
			proto.SetExtension(f.Service[0].Method[0].Options, annotations.E_Http, &annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/apis/v2/different"}, Body: "*"})
		}, "legacy HTTP binding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := contractFixture(t, "v2")
			tc.change(file)
			current, err := protodesc.NewFile(file, nil)
			require.NoError(t, err)
			err = legacyFileCompatible(legacy, current)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}

func TestLegacyContractNestedMessagesAndOneofs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		change    func(*descriptorpb.FileDescriptorProto)
		wantError string
	}{
		{"nested field removal", func(f *descriptorpb.FileDescriptorProto) { f.MessageType[0].NestedType[0].Field = nil }, "legacy field"},
		{"nested message removal", func(f *descriptorpb.FileDescriptorProto) { f.MessageType[0].NestedType = nil }, "legacy message"},
		{"oneof rename", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType[0].OneofDecl[0].Name = proto.String("different")
		}, "legacy oneof"},
		{"additional oneof before existing", func(f *descriptorpb.FileDescriptorProto) {
			f.MessageType[0].OneofDecl = append([]*descriptorpb.OneofDescriptorProto{{Name: proto.String("new_choice")}}, f.MessageType[0].OneofDecl...)
			f.MessageType[0].Field[0].OneofIndex = proto.Int32(1)
			f.MessageType[0].Field = append(f.MessageType[0].Field, &descriptorpb.FieldDescriptorProto{Name: proto.String("extra"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), OneofIndex: proto.Int32(0)})
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old, current := contractFixture(t, "v2beta1"), contractFixture(t, "v2")
			for _, file := range []*descriptorpb.FileDescriptorProto{old, current} {
				message := file.MessageType[0]
				message.NestedType = []*descriptorpb.DescriptorProto{{Name: proto.String("Nested"), Field: []*descriptorpb.FieldDescriptorProto{proto.Clone(message.Field[0]).(*descriptorpb.FieldDescriptorProto)}}}
				message.OneofDecl = []*descriptorpb.OneofDescriptorProto{{Name: proto.String("choice")}}
				message.Field[0].OneofIndex = proto.Int32(0)
			}
			tc.change(current)
			previous, err := protodesc.NewFile(old, nil)
			require.NoError(t, err)
			next, err := protodesc.NewFile(current, nil)
			require.NoError(t, err)
			err = legacyFileCompatible(previous, next)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}
