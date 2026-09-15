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
import { Alert, Box, Button, ButtonBase, Typography } from '@mui/material';
import ArrowBack from '@mui/icons-material/ArrowBack';
import Refresh from '@mui/icons-material/Refresh';
import RestartAlt from '@mui/icons-material/RestartAlt';
import { useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { V2beta1ArtifactTask } from 'src/apisv2beta1/artifact';
import { PipelineTaskTaskType } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { isInputArtifactTaskType, isOutputArtifactTaskType } from 'src/lib/v2/ArtifactTaskUtils';
import { getArtifactTypeName } from 'src/lib/v2/RuntimeArtifactUtils';
import { getTaskDisplayName } from 'src/lib/v2/RunTaskUtils';
import { RoutePageFactory } from 'src/components/Router';
import NativeLineageCanvas, { LineageEdge } from './NativeLineageCanvas';

const PAGE_SIZE = 5;
const controlStyle = {
  border: 0,
  background: 'transparent',
  boxShadow: 'none',
  textTransform: 'none',
  fontSize: 12,
} as const;
const labelStyle = {
  fontSize: 10,
  fontWeight: 600,
  letterSpacing: '.08em',
  textTransform: 'uppercase',
} as const;

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
        <Typography role='status' sx={{ fontSize: 12, p: 1 }}>
          Loading relationships…
        </Typography>
      )}
      {query.isError && (
        <Alert severity='warning'>
          Some relationships could not be loaded.
          <Button sx={controlStyle} onClick={() => query.refetch()}>
            Retry relationships
          </Button>
        </Alert>
      )}
      {children(rows)}
      {!query.isPending && !query.isError && !rows.length && (
        <Typography sx={{ fontSize: 12, p: 1 }}>No recorded relationships.</Typography>
      )}
      {repeated && (
        <Alert severity='warning'>The service repeated a page token. Refresh to try again.</Alert>
      )}
      {query.hasNextPage && (
        <Button
          aria-label='Load more relationships'
          sx={controlStyle}
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
    <Box>
      <ButtonBase
        data-lineage-node={nodeId}
        aria-disabled={selected || undefined}
        aria-current={selected ? 'location' : undefined}
        onClick={selected ? undefined : () => onSelect(id)}
        aria-label={name}
        title={`${name}\nArtifact ${id}${query.data?.uri ? `\n${query.data.uri}` : ''}`}
        sx={{
          display: 'block',
          width: '100%',
          minHeight: 108,
          textAlign: 'left',
          background: '#fff',
          color: '#20364d',
          border: selected ? '2px solid #2383e2' : '1px solid #cbd5df',
          borderRadius: '6px',
          padding: 0,
          boxShadow: selected ? '0 0 0 4px #2383e210' : '0 2px 4px #20364d08',
          '&:hover': { borderColor: '#2383e2', background: '#fafdff' },
          '&:focus-visible': { outline: '3px solid #2383e2', outlineOffset: 3 },
        }}
      >
        <Box sx={{ px: 1.75, py: 1.25, borderBottom: '1px solid #e8edf2' }}>
          <Typography
            sx={{
              ...labelStyle,
              color: '#39769f',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {query.data ? getArtifactTypeName(query.data) : 'Artifact'}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 1.75, py: 1.5 }}>
          <Box
            aria-hidden='true'
            sx={{
              width: 12,
              height: 12,
              flexShrink: 0,
              borderRadius: '50%',
              border: selected ? '3px solid #2383e2' : '1.5px solid #aebdca',
              background: selected ? '#e8f3ff' : 'transparent',
            }}
          />
          <Typography
            sx={{ fontSize: 14, fontWeight: selected ? 600 : 500, overflowWrap: 'anywhere' }}
          >
            {name}
          </Typography>
        </Box>
      </ButtonBase>
      {query.isError && (
        <Typography role='status' sx={{ fontSize: 12 }}>
          Details unavailable.{' '}
          <Button sx={controlStyle} onClick={() => query.refetch()}>
            Retry artifact
          </Button>
        </Typography>
      )}
    </Box>
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
    <Box
      component='section'
      aria-label={`${producer ? 'Producer' : 'Consumer'} ${name}`}
      sx={{
        display: 'grid',
        gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)',
        columnGap: '48px',
        alignItems: 'start',
        mb: 3,
      }}
    >
      <Box sx={{ gridColumn: producer ? 2 : 1, gridRow: 1 }}>
        <Box
          data-lineage-node={nodeId}
          sx={{
            background: '#294a68',
            minHeight: 108,
            color: '#fff',
            border: '1px solid #23425e',
            borderRadius: '6px',
            boxShadow: '0 2px 4px #20364d10',
            overflow: 'hidden',
          }}
        >
          <Typography
            sx={{
              ...labelStyle,
              color: '#93d5d2',
              px: 1.75,
              py: 1.25,
              borderBottom: '1px solid #ffffff20',
            }}
          >
            {task.data?.type === PipelineTaskTaskType.ROOT ? 'Run boundary' : 'Task'}
          </Typography>
          <Box sx={{ px: 1.75, py: 1.5, minHeight: 52 }}>
            {runId ? (
              <Link
                title={`Task ${taskId}\nRun ${runId}`}
                to={RoutePageFactory.runDetailsTask(runId, taskId)}
                style={{ color: '#fff', fontSize: 14, fontWeight: 500, overflowWrap: 'anywhere' }}
              >
                {name}
              </Link>
            ) : (
              <Typography sx={{ fontSize: 14 }}>{name}</Typography>
            )}
          </Box>
        </Box>
        <Typography
          title={relationship.key || '(unnamed port)'}
          sx={{
            color: '#647789',
            fontSize: 11,
            mt: 0.75,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {producer ? 'Output' : 'Input'} · {relationship.key || '(unnamed port)'}
        </Typography>
        <Button
          sx={{ ...controlStyle, px: 0, color: '#53728c', fontSize: 11 }}
          onClick={() => setExpanded(!expanded)}
          aria-expanded={expanded}
        >
          {expanded ? 'Hide' : 'Show'} {producer ? 'input' : 'output'} artifacts
        </Button>
        {task.isError && (
          <Typography role='status' sx={{ fontSize: 12 }}>
            Task details unavailable.{' '}
            <Button sx={controlStyle} onClick={() => task.refetch()}>
              Retry task
            </Button>
          </Typography>
        )}
      </Box>
      <LineageEdge
        id={nodeId}
        from={producer ? nodeId : targetNode}
        to={producer ? targetNode : nodeId}
      />
      <Box sx={{ gridColumn: producer ? 1 : 2, gridRow: 1 }}>
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
                      <Box key={relationshipId(row)} sx={{ mb: 2.5 }}>
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
                          <Typography sx={{ fontSize: 12 }}>Missing artifact identity</Typography>
                        )}
                        <Typography
                          title={row.key || '(unnamed port)'}
                          sx={{
                            mt: 0.75,
                            fontSize: 11,
                            color: '#647789',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {row.key || '(unnamed port)'}
                        </Typography>
                      </Box>
                    );
                  })}
                  {!adjacent.length && (
                    <Typography sx={{ color: '#8b99a6', fontSize: 12, pt: 5 }}>
                      No {producer ? 'inputs' : 'outputs'} loaded.
                    </Typography>
                  )}
                </>
              );
            }}
          </RelationshipPages>
        )}
      </Box>
    </Box>
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
    <ButtonBase
      aria-current={current ? 'location' : undefined}
      aria-disabled={current || undefined}
      onClick={current ? undefined : onClick}
      title={id}
      sx={{
        border: 0,
        background: 'transparent',
        borderRadius: 0,
        px: 0.5,
        py: 1,
        maxWidth: 220,
        color: current ? '#20364d' : '#3476ae',
        fontSize: 13,
        fontWeight: current ? 600 : 400,
        '&:focus-visible': { outline: '2px solid #2383e2' },
      }}
    >
      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {artifact.data?.name || id}
      </span>
    </ButtonBase>
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
    <Box>
      <Box
        sx={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 1,
          px: 2.5,
          py: 1,
        }}
      >
        <Box
          component='nav'
          aria-label='Lineage history'
          sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 0.5 }}
        >
          <Button
            aria-label='Back'
            sx={controlStyle}
            disabled={history.length === 1}
            onClick={() => setHistory((previous) => previous.slice(0, -1))}
          >
            <ArrowBack fontSize='small' />
          </Button>
          {history.map((id, index) => (
            <React.Fragment key={`${id}:${index}`}>
              {index > 0 && (
                <span aria-hidden='true' style={{ color: '#9aabb9' }}>
                  ›
                </span>
              )}
              <HistoryItem
                id={id}
                namespace={namespace}
                current={index === history.length - 1}
                onClick={() => setHistory((previous) => previous.slice(0, index + 1))}
              />
            </React.Fragment>
          ))}
          <span
            role='status'
            style={{
              position: 'absolute',
              width: 1,
              height: 1,
              overflow: 'hidden',
              clipPath: 'inset(50%)',
            }}
          >
            Neighborhood {history.length}
          </span>
        </Box>
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <Button
            startIcon={<Refresh />}
            sx={controlStyle}
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
            Refresh lineage
          </Button>
          <Button
            startIcon={<RestartAlt />}
            sx={controlStyle}
            disabled={history.length === 1}
            onClick={() => setHistory([artifactId])}
          >
            Reset
          </Button>
        </Box>
      </Box>
      <NativeLineageCanvas>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(5, minmax(0, 1fr))',
            gap: '48px',
            mb: 3,
          }}
        >
          {[
            'Input artifacts',
            'Producing tasks',
            'Selected artifact',
            'Consuming tasks',
            'Output artifacts',
          ].map((label) => (
            <Typography
              key={label}
              sx={{
                ...labelStyle,
                color: label === 'Selected artifact' ? '#2383e2' : '#718394',
                fontSize: 11,
              }}
            >
              {label}
            </Typography>
          ))}
        </Box>
        <RelationshipPages key={target} kind='artifact' id={target} namespace={namespace}>
          {(rows) => (
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: 'repeat(5, minmax(0, 1fr))',
                columnGap: '48px',
                alignItems: 'start',
              }}
            >
              <Box sx={{ gridColumn: 3, gridRow: 1 }}>
                <ArtifactNode
                  id={target}
                  nodeId={targetNode}
                  namespace={namespace}
                  onSelect={select}
                  selected
                />
              </Box>
              {[true, false].map((producer) => (
                <Box
                  key={String(producer)}
                  component='section'
                  aria-label={producer ? 'Producing tasks' : 'Consuming tasks'}
                  sx={{ gridColumn: producer ? '1 / 3' : '4 / 6', gridRow: 1 }}
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
                        <Typography key={relationshipId(row)}>Missing task identity</Typography>
                      ),
                    )}
                </Box>
              ))}
              {rows.some(
                (row) => !isInputArtifactTaskType(row.type) && !isOutputArtifactTaskType(row.type),
              ) && (
                <Alert severity='info' sx={{ gridColumn: '1 / -1', mt: 2 }}>
                  Some relationships have an unspecified direction and are not drawn.
                </Alert>
              )}
            </Box>
          )}
        </RelationshipPages>
      </NativeLineageCanvas>
      <Typography sx={{ px: 3, py: 1.5, fontSize: 12, color: '#718394' }}>
        Select an artifact to follow its lineage. Task names open Run Details. Only the loaded
        neighborhood is shown; use Load more for additional relationships.
      </Typography>
    </Box>
  );
}
