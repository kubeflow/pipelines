// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
)

type UUIDGeneratorInterface interface {
	NewRandom() (uuid.UUID, error)
}

// deterministicUUIDNamespace is a fixed namespace UUID used to derive stable
// (version 5) UUIDs from a name. It must never change; otherwise previously
// generated deterministic UUIDs would no longer match.
var deterministicUUIDNamespace = uuid.MustParse("c2f3a9d4-1e6b-4c8a-9f7d-0b5e3a1c2d4f")

// NewDeterministicUUID returns a stable UUID derived from the given name.
// Repeated calls with the same name always return the same UUID, which lets
// concurrent callers converge on a single primary key for idempotent inserts.
func NewDeterministicUUID(name string) string {
	return uuid.NewSHA1(deterministicUUIDNamespace, []byte(name)).String()
}

// UUIDGenerator is the concrete implementation of the UUIDGeneratorInterface used to
// generate UUIDs in production deployments.
type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (r *UUIDGenerator) NewRandom() (uuid.UUID, error) {
	return uuid.NewRandom()
}

// FakeUUIDGenerator is a fake implementation of the UUIDGeneratorInterface used for testing.
// It always generates the UUID and error provided during instantiation.
type FakeUUIDGenerator struct {
	uuidToReturn uuid.UUID
	errToReturn  error
}

// NewFakeUUIDGeneratorOrFatal creates a UUIDGenerator that always returns the UUID and error
// provided as parameters.
func NewFakeUUIDGeneratorOrFatal(uuidStringToReturn string, errToReturn error) UUIDGeneratorInterface {
	uuidToReturn, err := uuid.Parse(uuidStringToReturn)
	if err != nil {
		glog.Fatalf("Could not parse the UUID %v: %+v", uuidStringToReturn, err)
	}
	return &FakeUUIDGenerator{
		uuidToReturn: uuidToReturn,
		errToReturn:  errToReturn,
	}
}

func (f *FakeUUIDGenerator) NewRandom() (uuid.UUID, error) {
	return f.uuidToReturn, f.errToReturn
}
