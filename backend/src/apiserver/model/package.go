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

import (
	"fmt"
)

// PackageStatus a label for the status of the Package.
// This is intend to make package creation and deletion atomic.
type PackageStatus string

const (
	PackageCreating PackageStatus = "CREATING"
	PackageReady    PackageStatus = "READY"
	PackageDeleting PackageStatus = "DELETING"
)

type Package struct {
	UUID           string        `gorm:"column:UUID; not null; primary_key"`
	CreatedAtInSec int64         `gorm:"column:CreatedAtInSec; not null"`
	Name           string        `gorm:"column:Name; not null"`
	Description    string        `gorm:"column:Description; not null"`
	Parameters     string        `gorm:"column:Parameters; not null size:10000"` /* Json format argo.v1alpha1.parameter. Set max size to 10,000 */
	Status         PackageStatus `gorm:"column:Status; not null"`
}

func (p Package) GetValueOfPrimaryKey() string {
	return fmt.Sprint(p.UUID)
}

func GetPackageTablePrimaryKeyColumn() string {
	return "UUID"
}
