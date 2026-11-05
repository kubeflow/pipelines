// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientmanager

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMLMDNativeBootstrapRecoversInterruptedMySQL(t *testing.T) {
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to a disposable MySQL service")
	}
	for _, prefix := range []struct {
		name  string
		model any
	}{
		{name: "before tasks", model: &model.Run{}},
		{name: "before artifact tasks", model: &model.Artifact{}},
	} {
		t.Run(prefix.name, func(t *testing.T) {
			ctx := context.Background()
			sqlDB, _ := releasedStartupDatabase(t, dsn)
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, DefaultStringSize: 255}), &gorm.Config{})
			require.NoError(t, err)
			applyNativeModelPrefix(t, db, prefix.model)
			_, err = bootstrapNativeSchema(ctx, db, "mysql")
			require.Error(t, err, "unmarked partial schema must fail closed")
			require.NoError(t, db.Exec("CREATE TABLE "+nativeBootstrapMarker+" (id integer PRIMARY KEY)").Error)
			type result struct {
				resumed bool
				err     error
			}
			results := make(chan result, 2)
			for range 2 {
				go func() {
					resumed, err := bootstrapNativeSchema(ctx, db, "mysql")
					results <- result{resumed: resumed, err: err}
				}()
			}
			first, second := <-results, <-results
			require.NoError(t, first.err)
			require.NoError(t, second.err)
			require.NotEqual(t, first.resumed, second.resumed, "only one replica may resume bootstrap")
			state, err := mlmd.ClassifyStartup(ctx, sqlDB, "mysql")
			require.NoError(t, err)
			require.Equal(t, mlmd.Native, state)
			require.False(t, db.Migrator().HasTable(nativeBootstrapMarker))
		})
	}
}

func applyNativeModelPrefix(t *testing.T, db *gorm.DB, lastModel any) {
	t.Helper()
	models := model.AllModels()
	for i, entry := range models {
		if reflect.TypeOf(entry) == reflect.TypeOf(lastModel) {
			require.NoError(t, db.AutoMigrate(models[:i+1]...))
			return
		}
	}
	t.Fatalf("missing bootstrap test model %T", lastModel)
}
