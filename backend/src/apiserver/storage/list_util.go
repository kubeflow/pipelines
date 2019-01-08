// Copyright 2018 Google LLC
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

import (
	"database/sql"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func ScanRowToTotalSize(rows *sql.Rows) (int, error) {
	var total_size int
	rows.Next()
	err := rows.Scan(&total_size)
	if err != nil {
		return 0, util.NewInternalServerError(err, "Failed to scan row total_size")
	}
	return total_size, nil
}
