// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package model

// RecurringRunAdoption records completion of a one-time scheduling-state adoption.
// The receipt survives deletion of the adopted jobs and prevents reseeding them.
type RecurringRunAdoption struct {
	ID           string    `gorm:"column:ID; not null; primaryKey; type:varchar(191);"`
	AdoptedCount int64     `gorm:"column:AdoptedCount; not null;"`
	CompletedAt  int64     `gorm:"column:CompletedAt; not null;"`
	JobIDs       LargeText `gorm:"column:JobIDs; not null;"`
	Ready        bool      `gorm:"column:Ready; not null; default:false;"`
}

func (RecurringRunAdoption) TableName() string {
	return "recurring_run_adoptions"
}
