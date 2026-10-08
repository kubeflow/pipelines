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
import { ArrowLeft, RefreshCw, RotateCcw } from 'lucide-react';
import { Button } from './ui/button';
import { Alert } from './ui/alert';
import './NativeArtifactLineage.css';
import { useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router';
import { V2beta1ArtifactTask } from 'src/apisv2beta1/artifact';
import { PipelineTaskTaskType } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { isInputArtifactTaskType, isOutputArtifactTaskType } from 'src/lib/v2/ArtifactTaskUtils';
import { getArtifactTypeName } from 'src/lib/v2/RuntimeArtifactUtils';
import { getTaskDisplayName } from 'src/lib/v2/RunTaskUtils';
import { RoutePageFactory } from 'src/components/Router';
import NativeLineageCanvas, { LineageEdge } from './NativeLineageCanvas';

const PAGE_SIZE = 5;

function useArtifact(id: string, namespace: string) {
  return useQuery({
    queryKey: ['native-lineage-artifact', namespace, id],
    queryFn: ({ signal }) => Apis.artifactServiceApiV2.artifact_1(id, { signal }),
    staleTime: 60_000,
    retry: false,
  });
}

function relationshipId(row: V2beta1ArtifactTask) {
  return row.id || JSON.stringify([row.artifact_id, row.task_id, row.type, row.key]);
}

// Page one of a visible neighborhood is bounded. Further pages require user action.
function RelationshipPages({
  kind,
  id,
  namespace,
  children,
}: {
  kind: 'artifact' | 'task';
  id: string;
  namespace: string;
  children: (rows: V2beta1ArtifactTask[]) => React.ReactNode;
}) {
  const query = useInfiniteQuery({
    queryKey: ['native-lineage', namespace, kind, id],
    initialPageParam: '',
    queryFn: ({ pageParam, signal }) =>
      Apis.artifactServiceApiV2.artifactTasks(
        kind === 'task' ? [id] : undefined,
        undefined,
        kind === 'artifact' ? [id] : undefined,
        undefined,
        pageParam || undefined,
        PAGE_SIZE,
        undefined,
        undefined,
        { signal },
      ),
    getNextPageParam: (last, _pages, _previous, parameters) => {
      const next = last.next_page_token;
      return next && !parameters.includes(next) ? next : undefined;
    },
    staleTime: 60_000,
    retry: false,
  });
  const pages = query.data?.pages || [];
  const rows = Array.from(
    new Map(
      pages.flatMap((page) => page.artifact_tasks || []).map((row) => [relationshipId(row), row]),
    ).values(),
  );
  const token = pages.at(-1)?.next_page_token;
  const repeated = !!token && !!query.data?.pageParams.includes(token);
  return (
    <>
      {query.isPending && (
        <p role='status' className='kfp-lineage-status'>
          Loading relationships…
        </p>
      )}
      {query.isError && (
        <Alert variant='warning'>
          Some relationships could not be loaded.
          <Button variant='ghost' size='sm' onClick={() => query.refetch()}>
            Retry relationships
          </Button>
        </Alert>
      )}
      {children(rows)}
      {!query.isPending && !query.isError && !rows.length && (
        <p className='kfp-lineage-status'>No recorded relationships.</p>
      )}
      {repeated && (
        <Alert variant='warning'>The service repeated a page token. Refresh to try again.</Alert>
      )}
      {query.hasNextPage && (
        <Button
          variant='ghost'
          size='sm'
          aria-label='Load more relationships'
          disabled={query.isFetching}
          onClick={() => query.fetchNextPage()}
        >
          Load more
        </Button>
      )}
    </>
  );
}

function ArtifactNode({
  id,
  nodeId,
  namespace,
  onSelect,
  selected = false,
}: {
  id: string;
  nodeId: string;
  namespace: string;
  onSelect: (id: string) => void;
  selected?: boolean;
}) {
  const query = useArtifact(id, namespace);
  const name = query.data?.name || id;
  return (
    <div>
      <button
        type='button'
        data-lineage-node={nodeId}
        className='kfp-lineage-artifact'
        aria-disabled={selected || undefined}
        aria-current={selected ? 'location' : undefined}
        onClick={selected ? undefined : () => onSelect(id)}
        aria-label={name}
        title={`${name}\nArtifact ${id}${query.data?.uri ? `\n${query.data.uri}` : ''}`}
      >
        <span className='kfp-lineage-kind'>
          {query.data ? getArtifactTypeName(query.data) : 'Artifact'}
        </span>
        <span className='kfp-lineage-artifact-name'>{name}</span>
      </button>
      {query.isError && (
        <p role='status' className='kfp-lineage-status'>
          Details unavailable.{' '}
          <Button variant='ghost' size='sm' onClick={() => query.refetch()}>
            Retry artifact
          </Button>
        </p>
      )}
    </div>
  );
}

function TaskBranch({
  relationship,
  namespace,
  onSelect,
  targetNode,
}: {
  relationship: V2beta1ArtifactTask;
  namespace: string;
  onSelect: (id: string) => void;
  targetNode: string;
}) {
  const producer = isOutputArtifactTaskType(relationship.type);
  const taskId = relationship.task_id!;
  const runId = relationship.run_id || '';
  const nodeId = `task:${relationshipId(relationship)}`;
  const task = useQuery({
    queryKey: ['native-lineage-task', namespace, runId, taskId],
    queryFn: ({ signal }) => Apis.runServiceApiV2.task_1(runId, taskId, { signal }),
    enabled: !!runId,
    staleTime: 60_000,
    retry: false,
  });
  const name =
    task.data?.type === PipelineTaskTaskType.ROOT
      ? 'Run output (root)'
      : task.data
        ? getTaskDisplayName(task.data, taskId)
        : relationship.producer?.task_name || taskId;
  // First bounded page of visible tasks only: adjacent artifact nodes never recursively fetch.
  const [expanded, setExpanded] = React.useState(true);
  return (
    <section
      aria-label={`${producer ? 'Producer' : 'Consumer'} ${name}`}
      className='kfp-lineage-branch'
    >
      <div style={{ gridColumn: producer ? 2 : 1, gridRow: 1 }}>
        <div data-lineage-node={nodeId} className='kfp-lineage-task'>
          <div className='kfp-lineage-kind'>
            {task.data?.type === PipelineTaskTaskType.ROOT ? 'Run boundary' : 'Task'}
          </div>
          <div className='kfp-lineage-task-name'>
            {runId ? (
              <Link
                title={`Task ${taskId}\nRun ${runId}`}
                to={RoutePageFactory.runDetailsTask(runId, taskId)}
              >
                {name}
              </Link>
            ) : (
              name
            )}
          </div>
        </div>
        <p title={relationship.key || '(unnamed port)'} className='kfp-lineage-port'>
          {producer ? 'Output' : 'Input'} · {relationship.key || '(unnamed port)'}
        </p>
        <Button
          variant='ghost'
          size='sm'
          onClick={() => setExpanded(!expanded)}
          aria-expanded={expanded}
        >
          {expanded ? 'Hide' : 'Show'} {producer ? 'input' : 'output'} artifacts
        </Button>
        {task.isError && (
          <p role='status' className='kfp-lineage-status'>
            Task details unavailable.{' '}
            <Button variant='ghost' size='sm' onClick={() => task.refetch()}>
              Retry task
            </Button>
          </p>
        )}
      </div>
      <LineageEdge
        id={nodeId}
        from={producer ? nodeId : targetNode}
        to={producer ? targetNode : nodeId}
      />
      <div style={{ gridColumn: producer ? 1 : 2, gridRow: 1 }}>
        {expanded && (
          <RelationshipPages kind='task' id={taskId} namespace={namespace}>
            {(rows) => {
              const adjacent = rows.filter((row) =>
                producer ? isInputArtifactTaskType(row.type) : isOutputArtifactTaskType(row.type),
              );
              return (
                <>
                  {adjacent.map((row) => {
                    const artifactNode = `${nodeId}:artifact:${relationshipId(row)}`;
                    return (
                      <div key={relationshipId(row)} className='kfp-lineage-adjacent'>
                        {row.artifact_id ? (
                          <>
                            <ArtifactNode
                              id={row.artifact_id}
                              nodeId={artifactNode}
                              namespace={namespace}
                              onSelect={onSelect}
                            />
                            <LineageEdge
                              id={artifactNode}
                              from={producer ? artifactNode : nodeId}
                              to={producer ? nodeId : artifactNode}
                            />
                          </>
                        ) : (
                          <p className='kfp-lineage-status'>Missing artifact identity</p>
                        )}
                        <p title={row.key || '(unnamed port)'} className='kfp-lineage-port'>
                          {row.key || '(unnamed port)'}
                        </p>
                      </div>
                    );
                  })}
                  {!adjacent.length && (
                    <p className='kfp-lineage-status'>
                      No {producer ? 'inputs' : 'outputs'} loaded.
                    </p>
                  )}
                </>
              );
            }}
          </RelationshipPages>
        )}
      </div>
    </section>
  );
}

function HistoryItem({
  id,
  namespace,
  current,
  onClick,
}: {
  id: string;
  namespace: string;
  current: boolean;
  onClick: () => void;
}) {
  const artifact = useArtifact(id, namespace);
  return (
    <button
      type='button'
      className='kfp-lineage-history-item'
      aria-current={current ? 'location' : undefined}
      aria-disabled={current || undefined}
      onClick={current ? undefined : onClick}
      title={id}
    >
      <span>{artifact.data?.name || id}</span>
    </button>
  );
}

export default function NativeArtifactLineage({
  artifactId,
  namespace = '',
}: {
  artifactId: string;
  namespace?: string;
}) {
  const [history, setHistory] = React.useState([artifactId]);
  const queryClient = useQueryClient();
  const target = history[history.length - 1];
  const targetNode = `target:${target}`;
  const select = (id: string) => {
    if (id !== target) setHistory((previous) => [...previous, id]);
  };
  return (
    <div className='kfp-native-lineage'>
      <div className='kfp-lineage-toolbar'>
        <nav aria-label='Lineage history' className='kfp-lineage-history'>
          <Button
            variant='ghost'
            size='icon'
            aria-label='Back'
            disabled={history.length === 1}
            onClick={() => setHistory((previous) => previous.slice(0, -1))}
          >
            <ArrowLeft size={16} />
          </Button>
          {history.map((id, index) => (
            <React.Fragment key={`${id}:${index}`}>
              {index > 0 && <span aria-hidden='true'>›</span>}
              <HistoryItem
                id={id}
                namespace={namespace}
                current={index === history.length - 1}
                onClick={() => setHistory((previous) => previous.slice(0, index + 1))}
              />
            </React.Fragment>
          ))}
          <span role='status' className='kfp-shell-visually-hidden'>
            Neighborhood {history.length}
          </span>
        </nav>
        <div className='kfp-lineage-controls'>
          <Button
            variant='ghost'
            size='sm'
            onClick={() =>
              queryClient.invalidateQueries({
                predicate: (query) =>
                  query.queryKey[1] === namespace &&
                  ['native-lineage', 'native-lineage-artifact', 'native-lineage-task'].includes(
                    String(query.queryKey[0]),
                  ),
              })
            }
          >
            <RefreshCw size={16} /> Refresh lineage
          </Button>
          <Button
            variant='ghost'
            size='sm'
            disabled={history.length === 1}
            onClick={() => setHistory([artifactId])}
          >
            <RotateCcw size={16} /> Reset
          </Button>
        </div>
      </div>
      <NativeLineageCanvas>
        <div className='kfp-lineage-columns kfp-lineage-column-labels'>
          {[
            'Input artifacts',
            'Producing tasks',
            'Selected artifact',
            'Consuming tasks',
            'Output artifacts',
          ].map((label) => (
            <div key={label} data-selected={label === 'Selected artifact' || undefined}>
              {label}
            </div>
          ))}
        </div>
        <RelationshipPages key={target} kind='artifact' id={target} namespace={namespace}>
          {(rows) => (
            <div className='kfp-lineage-columns'>
              <div style={{ gridColumn: 3, gridRow: 1 }}>
                <ArtifactNode
                  id={target}
                  nodeId={targetNode}
                  namespace={namespace}
                  onSelect={select}
                  selected
                />
              </div>
              {[true, false].map((producer) => (
                <section
                  key={String(producer)}
                  aria-label={producer ? 'Producing tasks' : 'Consuming tasks'}
                  style={{ gridColumn: producer ? '1 / 3' : '4 / 6', gridRow: 1 }}
                >
                  {rows
                    .filter((row) =>
                      producer
                        ? isOutputArtifactTaskType(row.type)
                        : isInputArtifactTaskType(row.type),
                    )
                    .map((row) =>
                      row.task_id ? (
                        <TaskBranch
                          key={relationshipId(row)}
                          relationship={row}
                          namespace={namespace}
                          onSelect={select}
                          targetNode={targetNode}
                        />
                      ) : (
                        <p key={relationshipId(row)}>Missing task identity</p>
                      ),
                    )}
                </section>
              ))}
              {rows.some(
                (row) => !isInputArtifactTaskType(row.type) && !isOutputArtifactTaskType(row.type),
              ) && (
                <Alert variant='info' className='kfp-lineage-unknown'>
                  Some relationships have an unspecified direction and are not drawn.
                </Alert>
              )}
            </div>
          )}
        </RelationshipPages>
      </NativeLineageCanvas>
      <p className='kfp-lineage-help'>
        Select an artifact to follow its lineage. Task names open Run Details. Only the loaded
        neighborhood is shown; use Load more for additional relationships.
      </p>
    </div>
  );
}
