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

import { Folder } from 'lucide-react';
import { ArtifactFlowElementData, GRAPH_NODE_HEIGHT, GRAPH_NODE_WIDTH } from './Constants';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import './Graph.css';

interface ArtifactNodeProps {
  id: string;
  data: ArtifactFlowElementData;
}

export default function ArtifactNode({ id, data }: ArtifactNodeProps) {
  return (
    <>
      <button
        type='button'
        className='kfp-graph-node'
        style={{ width: GRAPH_NODE_WIDTH, height: GRAPH_NODE_HEIGHT }}
        data-tone={data.hasArtifact ? 'warning' : 'neutral'}
        title={data.label}
        aria-label={data.label}
        aria-describedby={`${id}-status`}
      >
        <Folder className='kfp-graph-type-icon' size={16} aria-hidden='true' />
        <span className='kfp-graph-node-copy'>
          <span className='kfp-graph-node-name' id={id} data-testid={id}>
            {data.label}
          </span>
          <span className='kfp-graph-node-meta' id={`${id}-status`}>
            {data.hasArtifact ? 'Artifact available' : 'Artifact'}
          </span>
        </span>
      </button>
      <ReadOnlyNodeHandles />
    </>
  );
}
