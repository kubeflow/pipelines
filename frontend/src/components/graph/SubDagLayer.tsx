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
import './Graph.css';

export interface SubDagLayerProps {
  layers: string[];
  onLayersUpdate(layers: string[]): void;
}

export default function SubDagLayer({ layers, onLayersUpdate }: SubDagLayerProps) {
  return (
    <nav className='kfp-graph-hierarchy' aria-label='Graph hierarchy'>
      <span className='kfp-graph-hierarchy-label'>Layers</span>
      <ol>
        {layers.map((layer, index) => {
          const active = index === layers.length - 1;
          return (
            <li key={`${index}-${layer}`}>
              {index > 0 && <ChevronRight size={14} aria-hidden='true' />}
              <button
                type='button'
                disabled={active}
                aria-current={active ? 'step' : undefined}
                onClick={() => onLayersUpdate(layers.slice(0, index + 1))}
              >
                {layer}
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
