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
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// LargeText is a custom data type defined per GORM's recommendation for dialect-aware
// large-text columns. It implements GormDBDataTypeInterface to return the appropriate
// SQL type for each dialect (e.g., LONGTEXT for MySQL, TEXT for others).
// For details, see https://gorm.io/docs/data_types.html#GormDataTypeInterface
type LargeText string

func (LargeText) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Name() {
	case "mysql":
		return "LONGTEXT"
	default:
		return "TEXT"
	}
}

func (lt LargeText) String() string {
	return string(lt)
}

func (lt LargeText) Value() (driver.Value, error) {
	return string(lt), nil
}

func (lt *LargeText) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*lt = LargeText(v)
	case []byte:
		*lt = LargeText(string(v))
	case nil:
		*lt = ""
	default:
		return fmt.Errorf("unsupported type %T for LargeText", v)
	}
	return nil
}

func (lt LargeText) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(lt))
}

func (lt *LargeText) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*lt = LargeText(s)
	return nil
}
