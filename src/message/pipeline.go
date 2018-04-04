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

package message

type Pipeline struct {
	*Metadata`json:",omitempty"`
	// TODO: change "-" to "createdAt", "updatedAt" once we manage these fields instead of GORM.
	CreatedAtInSec  int64       `json:"-" gorm:"not null"`
	UpdatedAtInSec  int64       `json:"-" gorm:"not null"`
	DeletedAtInSec  *int64      `json:"-" sql:"index"`
	Name            string      `json:"name" gorm:"not null"`
	Description     string      `json:"description,omitempty"`
	PackageId       uint        `json:"packageId" gorm:"not null"`
	Schedule        string      `json:"schedule" gorm:"not null"`
	Enabled         bool        `json:"enabled" gorm:"not null"`
	EnabledAtInSec  int64       `json:"enabledAt" gorm:"not null"`
	Parameters      []Parameter `json:"parameters,omitempty" gorm:"polymorphic:Owner;"`
}
