// Copyright 2021 The Kubeflow Authors
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

import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { Artifact, Execution } from 'src/third_party/mlmd';

export type SubDagKind = 'Sub-DAG' | 'Loop' | 'Iteration' | 'Condition';
export type FlowElementState = Execution.State | V2beta1RuntimeState;

// Being used as the base interface for Node and Edge in ReactFlow.
export type FlowElementDataBase = {
  label: string;
  mlmdId?: number;
  [key: string]: any;
};

export type SubDagTaskData = FlowElementDataBase & {
  groupKind?: SubDagKind;
  expectedTaskCount?: number;
  state?: FlowElementState;
};

export type SubDagFlowElementData = SubDagTaskData & {
  // Callback action if a SubDag expand button is clicked.
  expand: (nodeKey: string) => void;
};

export type ExecutionFlowElementData = FlowElementDataBase & {
  state?: FlowElementState;
};

export type ArtifactFlowElementData = FlowElementDataBase & {
  state?: Artifact.State;
  producerSubtask?: string;
  outputArtifactKey?: string;
  // Execution that produced the resolved artifact (mlmdId). Together with mlmdId it
  // uniquely identifies the OUTPUT event when an artifact carries more than one (e.g. a
  // same-run cache hit republishes an artifact under a new execution).
  producerExecutionId?: number;
};
