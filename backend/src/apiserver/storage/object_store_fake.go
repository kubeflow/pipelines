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

package storage

import "gocloud.dev/blob/memblob"

// Return the object store with blob storage for testing.
func NewFakeObjectStore() ObjectStoreInterface {
	// Use memory-based blob storage for testing
	bucket := memblob.OpenBucket(nil)
	return NewBlobObjectStore(bucket, "pipelines")
}
