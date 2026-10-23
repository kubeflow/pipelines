// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

// DriverTaskStop fences delayed creates even when the stopped driver never
// created a public task. StopKey includes the run generation and task identity.
type DriverTaskStop struct {
	StopKey string `gorm:"column:StopKey; primaryKey; type:varchar(64);"`
	RunUUID string `gorm:"column:RunUUID; not null; type:varchar(191); index:idx_driver_stop_run;"`
	Run     Run    `gorm:"foreignKey:RunUUID;references:UUID;constraint:driver_task_stops_RunUUID_run_details_UUID_foreign,OnDelete:CASCADE,OnUpdate:CASCADE;"`
}

func (DriverTaskStop) TableName() string { return "driver_task_stops" }
