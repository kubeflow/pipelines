// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package model

// TransferIdentity is the durable identity of an installation, shared by replicas.
type TransferIdentity struct {
	ID   uint8  `gorm:"column:ID;primaryKey;autoIncrement:false"`
	UUID string `gorm:"column:UUID;type:varchar(36);not null"`
}

// TransferReceipt records ownership and the original digest of a transferred resource.
// Key hashes source, namespace, kind and source ID to keep the index bounded.
type TransferReceipt struct {
	Key        string `gorm:"column:Key;type:varchar(64);primaryKey"`
	Source     string `gorm:"column:Source;type:varchar(36);not null"`
	Namespace  string `gorm:"column:Namespace;type:varchar(63);not null"`
	Kind       string `gorm:"column:Kind;type:varchar(32);not null"`
	SourceID   string `gorm:"column:SourceID;type:varchar(191);not null"`
	TargetID   string `gorm:"column:TargetID;type:varchar(191);not null"`
	Digest     string `gorm:"column:Digest;type:varchar(64);not null"`
	NamePrefix string `gorm:"column:NamePrefix;type:varchar(128);not null"`
}
