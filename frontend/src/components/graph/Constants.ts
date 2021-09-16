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

import { Execution, Artifact } from 'src/third_party/mlmd';

// Being used as the base interace for Node and Edge in Reactflow.
export type FlowElementDataBase = {
  label: string;
  [key: string]: any;
};

export type SubDagFlowElementData = FlowElementDataBase & {
  // Callback action if a SubDag expand button is clicked.
  expand: (nodeKey: string) => void;
};

export type ExecutionFlowElementData = FlowElementDataBase & {
  state?: Execution.State;
};

export type ArtifactFlowElementData = FlowElementDataBase & {
  state?: Artifact.State;
};
