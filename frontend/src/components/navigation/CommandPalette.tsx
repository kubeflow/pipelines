/*
 * Copyright 2026 The Kubeflow Authors
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

import { useEffect, useMemo, useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { Link } from 'react-router';
import { FlaskConical, PlayCircle, Search, Workflow } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { Apis } from 'src/lib/Apis';
import { encodeNameFilter } from 'src/lib/ApiFilter';
import { RoutePageFactory } from '../Router';
import { Button } from '../ui/button';
import { ModalDialog } from '../ui/dialog';
import { Input } from '../ui/input';
import type { AppShellNavItem } from '../shell/AppShell';
import './CommandPalette.css';

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  namespace?: string;
  requireNamespace?: boolean;
  items: readonly AppShellNavItem[];
}

interface SearchResult {
  id: string;
  name: string;
  href: string;
  archived?: boolean;
}

const resourceTypes = [
  { label: 'Runs', icon: PlayCircle },
  { label: 'Pipelines', icon: Workflow },
  { label: 'Experiments', icon: FlaskConical },
];

export function CommandPalette({ open, onClose, ...props }: CommandPaletteProps) {
  return (
    <ModalDialog open={open} onClose={onClose} title='Search and navigate'>
      {open && <PaletteSession {...props} onClose={onClose} />}
    </ModalDialog>
  );
}

function PaletteSession({
  onClose,
  namespace,
  requireNamespace = false,
  items,
}: Omit<CommandPaletteProps, 'open'>) {
  const [query, setQuery] = useState('');
  const [attempt, setAttempt] = useState(0);
  const [sharedPipelines, setSharedPipelines] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  // A new identity also distinguishes a return to an earlier query or namespace.
  const request = useMemo(
    () => ({ query: query.trim(), namespace, attempt, requireNamespace, sharedPipelines }),
    [query, namespace, attempt, requireNamespace, sharedPipelines],
  );
  const [response, setResponse] = useState<{
    request: typeof request;
    groups: PromiseSettledResult<SearchResult[]>[];
  }>();
  const needsNamespace = requireNamespace && !namespace;
  const canSearch = request.query.length >= 2 && !needsNamespace;
  const groups = canSearch && response?.request === request ? response.groups : undefined;
  const navigation = items.filter((item) =>
    item.label.toLowerCase().includes(request.query.toLowerCase()),
  );

  // External sync: debounce and cancel scoped HTTP reads on query changes and modal cleanup.
  useEffect(() => {
    if (!canSearch) return;
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      const filter = encodeNameFilter(request.query);
      const init = { signal: controller.signal };
      const results = await Promise.allSettled([
        Apis.runServiceApiV2
          .listRuns(
            request.namespace,
            undefined,
            undefined,
            5,
            'created_at desc',
            filter,
            true,
            undefined,
            init,
          )
          .then(({ runs }) =>
            (runs || [])
              .filter((run) => run.run_id)
              .slice(0, 5)
              .map((run) => ({
                id: run.run_id!,
                name: run.display_name || run.run_id!,
                href: RoutePageFactory.runDetails(run.run_id!),
                archived: run.storage_state === 'ARCHIVED',
              })),
          ),
        Apis.pipelineServiceApiV2
          .listPipelines(
            request.requireNamespace && request.sharedPipelines ? undefined : request.namespace,
            undefined,
            5,
            'created_at desc',
            filter,
            init,
          )
          .then(({ pipelines }) =>
            (pipelines || [])
              .filter((pipeline) => pipeline.pipeline_id)
              .slice(0, 5)
              .map((pipeline) => ({
                id: pipeline.pipeline_id!,
                name: pipeline.display_name || pipeline.name || pipeline.pipeline_id!,
                href: RoutePageFactory.pipelineDetails(pipeline.pipeline_id!),
              })),
          ),
        Apis.experimentServiceApiV2
          .listExperiments(undefined, 5, 'created_at desc', filter, request.namespace, init)
          .then(({ experiments }) =>
            (experiments || [])
              .filter((experiment) => experiment.experiment_id)
              .slice(0, 5)
              .map((experiment) => ({
                id: experiment.experiment_id!,
                name: experiment.display_name || experiment.experiment_id!,
                href: RoutePageFactory.experimentDetails(experiment.experiment_id!),
                archived: experiment.storage_state === 'ARCHIVED',
              })),
          ),
      ]);
      if (!controller.signal.aborted) setResponse({ request, groups: results });
    }, 275);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [canSearch, request]);

  function moveFocus(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return;
    const links = Array.from(
      containerRef.current?.querySelectorAll<HTMLAnchorElement>('[data-palette-result]') || [],
    );
    const controls = [inputRef.current, ...links].filter(
      (element): element is HTMLInputElement | HTMLAnchorElement => !!element,
    );
    const current = controls.indexOf(event.target as HTMLInputElement | HTMLAnchorElement);
    if (current < 0 || controls.length < 2) return;
    event.preventDefault();
    controls[
      (current + (event.key === 'ArrowDown' ? 1 : -1) + controls.length) % controls.length
    ].focus();
  }

  function renderResult(result: SearchResult, Icon: LucideIcon) {
    return (
      <li key={result.id}>
        <Link
          to={result.href}
          onClick={onClose}
          aria-label={result.archived ? `${result.name} (Archived)` : undefined}
          data-palette-result
          className='kfp-command-result'
        >
          <Icon size={17} strokeWidth={1.6} aria-hidden='true' />
          <span className='kfp-command-result-name'>{result.name}</span>
          {result.archived && <span className='kfp-command-archived'>Archived</span>}
        </Link>
      </li>
    );
  }

  const failedTypes =
    groups?.flatMap((group, index) =>
      group.status === 'rejected' ? [resourceTypes[index].label] : [],
    ) || [];
  const resultCount =
    groups?.reduce(
      (count, group) => count + (group.status === 'fulfilled' ? group.value.length : 0),
      0,
    ) || 0;

  return (
    <div className='kfp-command-palette' ref={containerRef} onKeyDown={moveFocus}>
      <div className='kfp-command-search'>
        <Search size={18} aria-hidden='true' />
        <Input
          ref={inputRef}
          type='search'
          aria-label='Search pipelines, experiments, and runs'
          placeholder='Search by name…'
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </div>
      <p className='kfp-command-scope'>
        {namespace
          ? `Namespace: ${namespace}`
          : needsNamespace
            ? 'Select a namespace to search resources.'
            : 'Search runs, pipelines, and experiments.'}
      </p>
      {requireNamespace && !needsNamespace && (
        <label className='kfp-command-pipeline-scope'>
          Pipeline scope
          <select
            value={sharedPipelines ? 'shared' : 'namespace'}
            onChange={(event) => setSharedPipelines(event.target.value === 'shared')}
          >
            <option value='namespace'>Current namespace</option>
            <option value='shared'>Shared pipelines</option>
          </select>
        </label>
      )}
      <div className='kfp-command-results'>
        {navigation.length > 0 && (
          <section aria-label='Navigation'>
            <h3>Go to</h3>
            <ul>
              {navigation.map((item) =>
                renderResult({ id: item.id, name: item.label, href: item.href }, item.icon),
              )}
            </ul>
          </section>
        )}
        <div role='status' className='kfp-command-status'>
          {!needsNamespace &&
            request.query.length < 2 &&
            'Type at least 2 characters to search resources.'}
          {canSearch && !groups && 'Searching…'}
          {canSearch &&
            groups &&
            resultCount > 0 &&
            `${resultCount} matching ${resultCount === 1 ? 'resource' : 'resources'} shown.`}
          {canSearch &&
            groups &&
            resultCount === 0 &&
            failedTypes.length === 0 &&
            'No matching resources.'}
        </div>
        {canSearch &&
          groups?.map(
            (group, index) =>
              group.status === 'fulfilled' &&
              group.value.length > 0 && (
                <section key={resourceTypes[index].label} aria-label={resourceTypes[index].label}>
                  <h3>{resourceTypes[index].label}</h3>
                  <ul>
                    {group.value.map((result) => renderResult(result, resourceTypes[index].icon))}
                  </ul>
                </section>
              ),
          )}
        {failedTypes.length > 0 && (
          <div className='kfp-command-error'>
            <p role='alert'>Could not search {failedTypes.join(', ')}.</p>
            <Button variant='secondary' size='sm' onClick={() => setAttempt((value) => value + 1)}>
              Retry search
            </Button>
          </div>
        )}
      </div>
      <footer className='kfp-command-footer'>
        <span>↑ ↓ or Tab to move · Enter to open</span>
        <Button variant='ghost' size='sm' onClick={onClose}>
          Close
        </Button>
      </footer>
    </div>
  );
}
