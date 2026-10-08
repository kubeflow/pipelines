/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { ChevronRight } from 'lucide-react';
import { SubDagFlowElementData, GRAPH_NODE_HEIGHT, GRAPH_NODE_WIDTH } from './Constants';
import { TaskStateIndicator } from './ExecutionNode';
import { getTaskStatus } from 'src/lib/StatusUtils';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import './Graph.css';

interface SubDagNodeProps {
  id: string;
  data: SubDagFlowElementData;
}

export default function SubDagNode({ id, data }: SubDagNodeProps) {
  const status = getTaskStatus(data.state);
  return (
    <>
      <div
        className='kfp-graph-node kfp-graph-group'
        style={{ width: GRAPH_NODE_WIDTH, height: GRAPH_NODE_HEIGHT }}
        data-tone={status.tone}
      >
        <button
          type='button'
          className='kfp-graph-group-select'
          title={data.label}
          aria-label={data.label}
          aria-describedby={`${id}-status`}
        >
          <TaskStateIndicator state={data.state} />
          <span className='kfp-graph-node-copy'>
            <span className='kfp-graph-node-name' id={id} data-testid={id}>
              {data.label}
            </span>
            <span className='kfp-graph-node-meta' id={`${id}-status`} aria-hidden='true'>
              {data.state === undefined ? 'Nested pipeline' : status.label}
            </span>
          </span>
        </button>
        <button
          type='button'
          className='kfp-graph-expand nodrag'
          aria-label={`Expand ${data.label}`}
          title={`Expand ${data.label}`}
          data-testid='expand-button'
          onClick={(event) => {
            event.stopPropagation();
            data.expand(id);
          }}
        >
          <ChevronRight size={16} aria-hidden='true' />
        </button>
      </div>
      <ReadOnlyNodeHandles />
    </>
  );
}
