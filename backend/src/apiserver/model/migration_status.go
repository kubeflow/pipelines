// Copyright 2026 The Kubeflow Authors
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

package model

// MigrationStatus records one-time data migrations that have already been
// applied, so that every apiserver replica does not re-run them on each
// startup. Schema migrations are handled by AutoMigrate; this table exists for
// data backfills, whose cost scales with retained history rather than with the
// schema.
type MigrationStatus struct {
	Name           string `gorm:"column:Name; not null; primaryKey; type:varchar(64);"`
	AppliedAtInSec int64  `gorm:"column:AppliedAtInSec; not null;"`
}

func (MigrationStatus) TableName() string {
	return "migration_statuses"
}
