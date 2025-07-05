package model

import "gorm.io/gorm"

// LongTextByDialect returns the correct database column type for large text
// fields according to the current GORM dialector.
// For MySQL it returns "longtext", and for PostgreSQL it returns "text".
func LongTextByDialect(db *gorm.DB) string {
	switch db.Name() {
	case "mysql":
		return "longtext"
	case "postgres", "pgx":
		return "text"
	}
	return ""
}
