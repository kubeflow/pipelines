// Copyright 2018 Google LLC
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

package storage

import (
	"database/sql"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func NullStringToPointer(ns sql.NullString) *string {
	if ns.Valid {
		return util.StringPointer(ns.String)
	}
	return nil
}

func NullInt64ToPointer(ni sql.NullInt64) *int64 {
	if ni.Valid {
		return util.Int64Pointer(ni.Int64)
	}
	return nil
}

func PointerToNullString(sp *string) sql.NullString {
	if sp == nil {
		return sql.NullString{}
	}
	return sql.NullString{
		String: *sp,
		Valid:  true,
	}
}

func PointerToNullInt64(ip *int64) sql.NullInt64 {
	if ip == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{
		Int64: *ip,
		Valid: true,
	}
}
