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

// RunLease retains its generation when released so an old owner cannot regain
// write authority through an owner-ID ABA. ExpiresAt is database Unix millis.
type RunLease struct {
	RunUUID    string  `gorm:"column:RunUUID;type:varchar(191);not null;primaryKey;index:idx_run_leases_expiry_run,priority:2"`
	OwnerID    *string `gorm:"column:OwnerID;type:varchar(191)"`
	Generation int64   `gorm:"column:Generation;not null;default:0"`
	ExpiresAt  *int64  `gorm:"column:ExpiresAt;index:idx_run_leases_expiry_run,priority:1"`
	Run        Run     `gorm:"foreignKey:RunUUID;references:UUID;constraint:fk_run_leases_run,OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// TableName returns the durable worker-ownership table.
func (RunLease) TableName() string { return "run_leases" }
