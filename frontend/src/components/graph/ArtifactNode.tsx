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
import { Tooltip } from '@base-ui/react/tooltip';
import { useTheme } from 'src/components/shell/ThemeProvider';
import { ArtifactFlowElementData, GRAPH_NODE_HEIGHT, GRAPH_NODE_WIDTH } from './Constants';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import './Graph.css';

interface ArtifactNodeProps {
  id: string;
  selected?: boolean;
  data: ArtifactFlowElementData;
}

export default function ArtifactNode({ id, data, selected }: ArtifactNodeProps) {
  const { themeClassName } = useTheme();
  const description = data.hasArtifact ? 'Artifact available' : 'Artifact';
  return (
    <>
      <Tooltip.Root>
        <Tooltip.Trigger
          render={<button type='button' />}
          delay={150}
          className='kfp-graph-node'
          data-selected={selected || undefined}
          style={{ width: GRAPH_NODE_WIDTH, height: GRAPH_NODE_HEIGHT }}
          data-tone={data.hasArtifact ? 'warning' : 'neutral'}
          aria-label={data.label}
          aria-describedby={`${id}-status`}
        >
          <Folder
            className='kfp-graph-type-icon'
            data-kind='artifact'
            size={16}
            role='img'
            aria-label='Artifact'
          />
          <span className='kfp-graph-node-copy'>
            <span className='kfp-graph-node-name' id={id} data-testid={id}>
              {data.label}
            </span>
          </span>
          <span className='kfp-graph-node-status' data-empty='true' aria-hidden='true' />
          <span className='sr-only' id={`${id}-status`}>
            {description}
          </span>
        </Tooltip.Trigger>
        <Tooltip.Portal className={themeClassName}>
          <Tooltip.Positioner side='top' sideOffset={6}>
            <Tooltip.Popup role='tooltip' className='kfp-graph-tooltip'>
              {data.label} · {description}
            </Tooltip.Popup>
          </Tooltip.Positioner>
        </Tooltip.Portal>
      </Tooltip.Root>
      <ReadOnlyNodeHandles />
    </>
  );
}
