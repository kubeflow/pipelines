package storage

import (
	"database/sql"

	"github.com/googleprivate/ml/backend/src/common/util"
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
