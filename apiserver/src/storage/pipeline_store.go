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

package storage

import (
	"fmt"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"

	"github.com/jinzhu/gorm"
)

type PipelineStoreInterface interface {
	ListPipelines() ([]pipelinemanager.Pipeline, error)
	GetPipeline(id uint) (pipelinemanager.Pipeline, error)
	CreatePipeline(pipelinemanager.Pipeline) (pipelinemanager.Pipeline, error)
}

type PipelineStore struct {
	db *gorm.DB
}

func (s *PipelineStore) ListPipelines() ([]pipelinemanager.Pipeline, error) {
	var pipelines []pipelinemanager.Pipeline
	// List the pipelines as well as their parameters.
	// Preload parameter table first to optimize DB transaction.
	if r := s.db.Preload("Parameters").Find(&pipelines); r.Error != nil {
		return nil, util.NewInternalError("Failed to list pipelines", r.Error.Error())
	}
	return pipelines, nil
}

func (s *PipelineStore) GetPipeline(id uint) (pipelinemanager.Pipeline, error) {
	var pipeline pipelinemanager.Pipeline
	// Get the pipeline as well as its parameter.
	if r := s.db.Preload("Parameters").First(&pipeline, id); r.Error != nil {
		// Error returns when no pipeline found.
		return pipeline, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	return pipeline, nil
}

func (s *PipelineStore) CreatePipeline(p pipelinemanager.Pipeline) (pipelinemanager.Pipeline, error) {
	r := s.db.Create(&p)
	if r.Error != nil {
		return p, util.NewInternalError("Failed to add pipeline to pipeline table", r.Error.Error())
	}
	return p, nil
}

// factory function for pipeline store
func NewPipelineStore(db *gorm.DB) *PipelineStore {
	return &PipelineStore{db: db}
}
