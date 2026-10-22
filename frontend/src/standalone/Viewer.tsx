/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { useEffect, useMemo, useState } from 'react';
import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import PipelineGraph from 'src/components/graph/PipelineGraph';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import type { LayerFlowElement } from 'src/lib/v2/FlowTypes';
import { convertFlowElements } from 'src/lib/v2/StaticFlow';

export interface ViewerState {
  spec: Record<string, unknown> | null;
  error: string | null;
  revision: number;
}

export function Viewer({ initial, live }: { initial: ViewerState; live: boolean }) {
  const [state, setState] = useState(initial);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const [selection, setSelection] = useState<LayerFlowElement | null>(null);

  // External synchronization with the loopback watch server; exports never fetch.
  useEffect(() => {
    if (!live) return;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try {
        const response = await fetch('./state', { signal: controller.signal, cache: 'no-store' });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const next: ViewerState = await response.json();
        if (controller.signal.aborted) return;
        setState((current) => (current.revision === next.revision ? current : next));
        setConnectionError(null);
      } catch {
        if (!controller.signal.aborted) {
          setConnectionError('Disconnected from watch server. Showing the last received graph.');
        }
      } finally {
        if (!controller.signal.aborted) timer = setTimeout(poll, 750);
      }
    };
    void poll();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [live]);

  const graph = useMemo(() => {
    if (!state.spec) return null;
    try {
      const spec = PipelineSpec.fromJSON(state.spec);
      return {
        name: spec.pipelineInfo?.name,
        template: JSON.stringify(state.spec),
        elements: convertFlowElements(spec),
      };
    } catch (error) {
      return { error: `Unable to display pipeline: ${String(error)}` };
    }
  }, [state.spec]);

  return (
    <ThemeProvider className='standalone-viewer'>
      <main className='standalone-viewer'>
        <header className='standalone-header'>
          <div>
            <span className='standalone-subtitle'>Kubeflow Pipelines · Local preview</span>
            <h1>{graph?.name ?? 'Waiting for a valid pipeline'}</h1>
          </div>
          <span>{live ? 'Watching saved changes' : 'Offline export'}</span>
        </header>
        {(state.error || connectionError || graph?.error) && (
          <div className='standalone-error' role='alert'>
            {state.error && (
              <>
                <strong>Compilation or export failed. Showing the last valid graph.</strong>
                <pre>{state.error}</pre>
              </>
            )}
            {connectionError && <p>{connectionError}</p>}
            {graph?.error && <p>{graph.error}</p>}
          </div>
        )}
        {graph?.elements && (
          <PipelineGraph
            templateString={graph.template}
            elements={graph.elements}
            selection={selection}
            onSelectionChange={setSelection}
          />
        )}
      </main>
    </ThemeProvider>
  );
}
