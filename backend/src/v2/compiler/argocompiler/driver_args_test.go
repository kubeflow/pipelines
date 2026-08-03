// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/driver/driverflags"
	"github.com/stretchr/testify/require"
)

type boolFlag interface {
	IsBoolFlag() bool
}

func assertRegisteredDriverArgs(t *testing.T, args []string) {
	t.Helper()

	fs := flag.NewFlagSet("driver", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	driverflags.RegisterDriverFlags(fs)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}

		flagName := strings.TrimPrefix(arg, "--")
		if eq := strings.Index(flagName, "="); eq >= 0 {
			flagName = flagName[:eq]
		}
		registeredFlag := fs.Lookup(flagName)
		require.NotNilf(t, registeredFlag, "compiler emits unregistered driver flag %q at index %d", arg, i)

		if boolValue, ok := registeredFlag.Value.(boolFlag); ok && boolValue.IsBoolFlag() {
			continue
		}
		if strings.Contains(arg, "=") {
			continue
		}
		if i+1 >= len(args) {
			t.Fatalf("compiler emits driver flag %q at index %d without required value", arg, i)
		}
		next := args[i+1]
		if strings.HasPrefix(next, "--") {
			nextFlagName := strings.TrimPrefix(next, "--")
			if eq := strings.Index(nextFlagName, "="); eq >= 0 {
				nextFlagName = nextFlagName[:eq]
			}
			if fs.Lookup(nextFlagName) != nil {
				t.Fatalf("compiler emits driver flag %q at index %d without required value before next flag %q", arg, i, next)
			}
		}
		i++
	}
}
