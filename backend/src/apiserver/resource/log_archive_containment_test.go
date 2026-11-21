// Copyright 2026 The Kubeflow Authors
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

package resource

import (
	"bytes"
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// This key is supplied by workflow history, not established by run authorization.
const foreignLogManifest = `{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"name":"native-run"},"status":{"nodes":{"node-id":{"outputs":{"artifacts":[{"name":"main-logs","s3":{"key":"tenant-b/private/main.log"}}]}}}}}`

func TestArchiveLogOwnershipContainment(t *testing.T) {
	previous := viper.Get(common.MultiUserMode)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	for _, multiuser := range []bool{false, true} {
		t.Run(map[bool]string{false: "native standalone preserved", true: "multiuser foreign key denied"}[multiuser], func(t *testing.T) {
			viper.Set(common.MultiUserMode, multiuser)
			objects := &readerOnlyObjectStore{files: map[string][]byte{"tenant-b/private/main.log": []byte("SYNTHETIC_FOREIGN_LOG\n")}}
			manager := &ResourceManager{objectStore: objects, logArchive: archive.NewLogArchive("/logs", "main.log")}
			var dst bytes.Buffer
			err := manager.readRunLogFromArchive(context.Background(), foreignLogManifest, "node-id", &dst)
			if multiuser {
				require.Error(t, err)
				assert.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "%v", err)
				assert.Empty(t, objects.getFileReaderPaths)
				assert.Empty(t, dst.String())
			} else {
				require.NoError(t, err)
				assert.Equal(t, []string{"tenant-b/private/main.log"}, objects.getFileReaderPaths)
				assert.Equal(t, "SYNTHETIC_FOREIGN_LOG\n", dst.String())
			}
		})
	}
}

func TestMultiUserArchiveDeniedBeforeManifestResolution(t *testing.T) {
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	// Nil archive/store and malformed history must not be consulted.
	manager := &ResourceManager{}
	var dst bytes.Buffer
	err := manager.readRunLogFromArchive(context.Background(), "invalid history", "node-id", &dst)
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "%v", err)
	assert.Empty(t, dst.String())
}
