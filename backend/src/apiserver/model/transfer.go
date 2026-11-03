// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package model

// TransferIdentity gives this database a stable installation identity.
type TransferIdentity struct {
	Key  string `gorm:"column:Key;primaryKey;type:varchar(32)"`
	UUID string `gorm:"column:UUID;not null;type:varchar(64)"`
}

// TransferReceipt records an immutable source resource and its destination ID.
// The hashed key includes source installation, namespace, kind and source ID.
type TransferReceipt struct {
	Key       string `gorm:"column:Key;primaryKey;type:varchar(64)"`
	Source    string `gorm:"column:Source;not null;type:varchar(64)"`
	Namespace string `gorm:"column:Namespace;not null;type:varchar(63)"`
	Kind      string `gorm:"column:Kind;not null;type:varchar(32)"`
	SourceID  string `gorm:"column:SourceID;not null;type:varchar(191)"`
	TargetID  string `gorm:"column:TargetID;not null;type:varchar(191)"`
	Digest    string `gorm:"column:Digest;not null;type:varchar(64)"`
}
