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

package common

// A deserialized token. Assuming the list request is sorted by name, a typical token should be
// {SortByFieldValue:"foo", KeyFieldValue:"2"}
// The corresponding list query would be
// select * from table where (name, id) >=(foobar,2) order by name, id limit page_size
type Token struct {
	// The value of the sorted field of the next row to be returned.
	SortByFieldValue string
	// The value of the key field of the next row to be returned.
	KeyFieldValue string
}

type PaginationContext struct {
	PageSize        int
	SortByFieldName string
	KeyFieldName    string
	IsDesc          bool
	Token           *Token
}
