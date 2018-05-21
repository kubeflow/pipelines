package model

// The data model that supports List APIs.
type ListableDataModel interface {
	// Get the value of the key field.
	GetValueOfPrimaryKey() string
}
