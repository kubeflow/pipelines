// Copyright 2025 The Kubeflow Authors
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
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// LargeText is a custom data type defined per GORM's recommendation for dialect-aware
// large-text columns. It implements GormDBDataTypeInterface to return the appropriate
// SQL type for each dialect (e.g., LONGTEXT for MySQL, TEXT for others).
// For details, see https://gorm.io/docs/data_types.html#GormDataTypeInterface
type LargeText string

func (LargeText) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql":
		return "LONGTEXT"
	default:
		return "TEXT"
	}
}
