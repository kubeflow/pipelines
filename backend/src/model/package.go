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

package model

// PackageStatus a label for the status of the Package.
// This is intend to make package creation and deletion atomic.
type PackageStatus string

const (
	PackageCreating PackageStatus = "CREATING"
	PackageReady    PackageStatus = "READY"
	PackageDeleting PackageStatus = "DELETING"
)

type Package struct {
	ID             uint   `gorm:"primary_key"`
	CreatedAtInSec int64  `gorm:"not null"`
	Name           string `gorm:"not null"`
	Description    string
	Parameters     []Parameter   `gorm:"polymorphic:Owner;"`
	Status         PackageStatus `gorm:"not null"`
}
