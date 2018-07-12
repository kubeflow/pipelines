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

package client

import (
	"github.com/googleprivate/ml/backend/src/common/util"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type PipelineClientFake struct {
	workflows          map[string]*util.Workflow
	scheduledWorkflows map[string]*util.ScheduledWorkflow
	err                error
}

func NewPipelineClientFake() *PipelineClientFake {
	return &PipelineClientFake{
		workflows:          make(map[string]*util.Workflow),
		scheduledWorkflows: make(map[string]*util.ScheduledWorkflow),
		err:                nil,
	}
}

func (p *PipelineClientFake) ReportWorkflow(workflow *util.Workflow) error {
	if p.err != nil {
		return p.err
	}
	p.workflows[getKey(workflow.Namespace, workflow.Name)] = workflow
	return nil
}

func (p *PipelineClientFake) ReportScheduledWorkflow(swf *util.ScheduledWorkflow) error {
	if p.err != nil {
		return p.err
	}
	p.scheduledWorkflows[getKey(swf.Namespace, swf.Name)] = swf
	return nil
}

func (p *PipelineClientFake) SetError(err error) {
	p.err = err
}

func (p *PipelineClientFake) GetWorkflow(namespace string, name string) *util.Workflow {
	return p.workflows[getKey(namespace, name)]
}

func (p *PipelineClientFake) GetScheduledWorkflow(namespace string, name string) *util.ScheduledWorkflow {
	return p.scheduledWorkflows[getKey(namespace, name)]
}
