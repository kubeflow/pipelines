// Copyright 2025 The Kubeflow Authors
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

package util

// ScopePath provides hierarchical navigation through a pipeline's DAG (Directed Acyclic Graph) structure.
// It maintains the execution context by tracking a path from the root component through nested tasks,
// storing each task's name, task spec, and component spec along the way. This allows the pipeline
// runtime to resolve inputs/outputs and understand the current position within nested DAG components.
//
// The path is implemented as a linked list, starting with "root" and growing as tasks are pushed onto it.
// It supports stack-like operations (Push/Pop) for traversing into and out of nested components.

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type ScopePath struct {
	list               *LinkedList[ScopePathEntry]
	pipelineSpec       *pipelinespec.PipelineSpec
	pipelineSpecStruct *structpb.Struct
	size               int
}
type ScopePathEntry struct {
	taskName      string
	taskSpec      *pipelinespec.PipelineTaskSpec
	componentSpec *pipelinespec.ComponentSpec
}

func (e *ScopePathEntry) GetTaskSpec() *pipelinespec.PipelineTaskSpec {
	return e.taskSpec
}

func (e *ScopePathEntry) GetComponentSpec() *pipelinespec.ComponentSpec {
	return e.componentSpec
}

func newScopePath(
	pipelineSpec *pipelinespec.PipelineSpec,
	pipelineSpecStruct *structpb.Struct,
) ScopePath {
	return ScopePath{
		pipelineSpec:       pipelineSpec,
		pipelineSpecStruct: pipelineSpecStruct,
	}
}

func NewScopePathFromStruct(spec *structpb.Struct) (ScopePath, error) {
	pipelineSpec := &pipelinespec.PipelineSpec{}
	// Convert struct to JSON
	b, err := spec.MarshalJSON()
	if err != nil {
		return ScopePath{}, fmt.Errorf("failed to marshal spec to JSON: %w", err)
	}
	// Unmarshal JSON to PipelineSpec
	if err := protojson.Unmarshal(b, pipelineSpec); err != nil {
		return ScopePath{}, fmt.Errorf("failed to unmarshal spec: %w", err)
	}
	return newScopePath(pipelineSpec, spec), nil
}

// ScopePathFromStringPathWithNewTask builds a ScopePath from a string path and pushes the newTask to the end of the path.
func ScopePathFromStringPathWithNewTask(rawPipelineSpec *structpb.Struct, path []string, newTask string) (ScopePath, error) {
	if rawPipelineSpec == nil {
		return ScopePath{}, fmt.Errorf("PipelineSpec is nil")
	}
	scopePath, err := ScopePathFromStringPath(rawPipelineSpec, path)
	if err != nil {
		return ScopePath{}, fmt.Errorf("failed to build scope path: %w", err)
	}
	// Update scope path to current context
	err = scopePath.Push(newTask)
	if err != nil {
		return ScopePath{}, err
	}
	return scopePath, nil
}

// ScopePathFromStringPath builds a ScopePath from a string path.
func ScopePathFromStringPath(rawPipelineSpec *structpb.Struct, path []string) (ScopePath, error) {
	if rawPipelineSpec == nil {
		return ScopePath{}, fmt.Errorf("PipelineSpec is nil")
	}
	scopePath, err := NewScopePathFromStruct(rawPipelineSpec)
	if err != nil {
		return ScopePath{}, fmt.Errorf("failed to build scope path: %w", err)
	}
	for _, taskName := range path {
		if err := scopePath.Push(taskName); err != nil {
			return ScopePath{}, fmt.Errorf("failed to build scope path at task %q: %w", taskName, err)
		}
	}
	return scopePath, nil
}

func (s *ScopePath) Push(taskName string) error {
	if s.list == nil {
		s.list = &LinkedList[ScopePathEntry]{}
	}
	if taskName == "root" {
		sp := ScopePathEntry{
			taskName:      taskName,
			componentSpec: s.pipelineSpec.Root,
		}
		s.list.append(sp)
		s.size++
		return nil
	}
	if s.list.head == nil {
		return fmt.Errorf("scope path is empty, first task should be root")
	}
	if s.list.head.Value.componentSpec.GetDag() == nil {
		return fmt.Errorf("this component is not a DAG component")
	}
	lastTask := s.GetLast()
	if lastTask == nil {
		return fmt.Errorf("last task is nil")
	}
	if _, ok := lastTask.componentSpec.GetDag().Tasks[taskName]; !ok {
		return fmt.Errorf("task %s is not found", taskName)
	}
	taskSpec := lastTask.componentSpec.GetDag().Tasks[taskName]
	if _, ok := s.pipelineSpec.Components[taskSpec.GetComponentRef().GetName()]; !ok {
		return fmt.Errorf("component %s is not found", taskSpec.GetComponentRef().GetName())
	}
	componentSpec := s.pipelineSpec.Components[taskSpec.GetComponentRef().GetName()]
	sp := ScopePathEntry{
		taskName:      taskName,
		taskSpec:      taskSpec,
		componentSpec: componentSpec,
	}
	s.list.append(sp)
	s.size++
	return nil
}

func (s *ScopePath) Pop() (ScopePathEntry, bool) {
	entry, ok := s.list.pop()
	if ok {
		s.size--
	}
	return entry, ok
}

func (s *ScopePath) GetRoot() *ScopePathEntry {
	return &s.list.head.Value
}

func (s *ScopePath) GetLast() *ScopePathEntry {
	spe, ok := s.list.last()
	if !ok {
		return nil
	}
	return &spe
}

func (s *ScopePath) GetSize() int {
	return s.size
}

func (s *ScopePath) GetPipelineSpec() *pipelinespec.PipelineSpec {
	return s.pipelineSpec
}

func (s *ScopePath) GetPipelineSpecStruct() *structpb.Struct {
	return s.pipelineSpecStruct
}

func (s *ScopePath) StringPath() []string {
	var path []string
	if s.list == nil {
		return path
	}
	for n := s.list.head; n != nil; n = n.Next {
		path = append(path, n.Value.taskName)
	}
	return path
}

// Node represents one element in the list.
type Node[T any] struct {
	Value T
	Next  *Node[T]
}

// LinkedList is a simple singly linked list.
type LinkedList[T any] struct {
	head *Node[T]
}

// append adds a new node to the end of the list.
func (l *LinkedList[T]) append(v T) {
	newNode := &Node[T]{Value: v}
	if l.head == nil {
		l.head = newNode
		return
	}
	curr := l.head
	for curr.Next != nil {
		curr = curr.Next
	}
	curr.Next = newNode
}

// pop removes and returns the last element.
// Returns (zeroValue, false) if list is empty.
func (l *LinkedList[T]) pop() (T, bool) {
	var zero T
	if l.head == nil {
		return zero, false
	}
	// Single element case
	if l.head.Next == nil {
		val := l.head.Value
		l.head = nil
		return val, true
	}
	// Traverse to second-last node
	curr := l.head
	for curr.Next.Next != nil {
		curr = curr.Next
	}
	val := curr.Next.Value
	curr.Next = nil
	return val, true
}

// last returns the value of the last node without removing it.
// Returns (zeroValue, false) if list is empty.
func (l *LinkedList[T]) last() (T, bool) {
	var zero T
	if l.head == nil {
		return zero, false
	}
	curr := l.head
	for curr.Next != nil {
		curr = curr.Next
	}
	return curr.Value, true
}
