// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as React from 'react';
import { Box } from '@mui/material';

interface Edge {
  id: string;
  from: string;
  to: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

/** DOM-backed edges follow async node sizing, pagination and browser resizing. */
export default function NativeLineageCanvas({ children }: { children: React.ReactNode }) {
  const ref = React.useRef<HTMLDivElement>(null);
  const [edges, setEdges] = React.useState<Edge[]>([]);
  const marker = React.useId().replace(/:/g, '');

  // External sync: measure actual DOM ports, not React-derived application state.
  React.useLayoutEffect(() => {
    const root = ref.current;
    if (!root) return;
    let frame = 0;
    const observed = new Set<Element>();
    const measure = () => {
      const origin = root.getBoundingClientRect();
      const nodes = new Map<string, DOMRect>();
      root.querySelectorAll<HTMLElement>('[data-lineage-node]').forEach((node) => {
        nodes.set(node.dataset.lineageNode!, node.getBoundingClientRect());
        if (!observed.has(node)) {
          observer.observe(node);
          observed.add(node);
        }
      });
      for (const node of observed)
        if (!root.contains(node)) {
          observer.unobserve(node);
          observed.delete(node);
        }
      const next: Edge[] = [];
      root.querySelectorAll<HTMLElement>('[data-lineage-edge]').forEach((edge) => {
        const from = edge.dataset.from!;
        const to = edge.dataset.to!;
        const source = nodes.get(from);
        const target = nodes.get(to);
        if (!source || !target) return;
        next.push({
          id: edge.dataset.lineageEdge!,
          from,
          to,
          x1: source.right - origin.left,
          y1: source.top + source.height / 2 - origin.top,
          x2: target.left - origin.left,
          y2: target.top + target.height / 2 - origin.top,
        });
      });
      setEdges((previous) => (JSON.stringify(previous) === JSON.stringify(next) ? previous : next));
    };
    const schedule = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(measure);
    };
    const observer = new ResizeObserver(schedule);
    observer.observe(root);
    const mutation = new MutationObserver((changes) => {
      if (changes.some((change) => !(change.target instanceof SVGElement))) schedule();
    });
    mutation.observe(root, { childList: true, subtree: true, characterData: true });
    schedule();
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      mutation.disconnect();
    };
  }, []);

  return (
    <Box
      role='region'
      aria-label='Lineage graph'
      tabIndex={0}
      sx={{
        overflowX: 'auto',
        background: '#f5f7fa',
        borderTop: '1px solid #e1e6ec',
        borderBottom: '1px solid #e1e6ec',
        '&:focus-visible': { outline: '2px solid #1a73e8', outlineOffset: -2 },
      }}
    >
      <Box
        ref={ref}
        sx={{ position: 'relative', minWidth: 1000, minHeight: 420, p: '24px 28px 40px' }}
      >
        <svg
          aria-hidden='true'
          style={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            pointerEvents: 'none',
            overflow: 'visible',
          }}
        >
          <defs>
            <marker id={marker} markerWidth='6' markerHeight='6' refX='5' refY='3' orient='auto'>
              <path d='M0,0 L6,3 L0,6' fill='#90a4b8' />
            </marker>
          </defs>
          {edges.map((edge) => (
            <g key={edge.id}>
              <path
                data-from={edge.from}
                data-to={edge.to}
                d={`M${edge.x1},${edge.y1} C${(edge.x1 + edge.x2) / 2},${edge.y1} ${(edge.x1 + edge.x2) / 2},${edge.y2} ${edge.x2},${edge.y2}`}
                fill='none'
                stroke='#90a4b8'
                strokeWidth='1.5'
                markerEnd={`url(#${marker})`}
              />
              <circle cx={edge.x1} cy={edge.y1} r='3' fill='#7891a8' />
            </g>
          ))}
        </svg>
        <Box sx={{ position: 'relative' }}>{children}</Box>
      </Box>
    </Box>
  );
}

export function LineageEdge({ id, from, to }: { id: string; from: string; to: string }) {
  return <span hidden data-lineage-edge={id} data-from={from} data-to={to} />;
}
