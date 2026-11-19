// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { useMemo, useState } from 'react';
import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  FormControlLabel,
  Paper,
  Stack,
  Typography,
} from '@mui/material';
import PipelineGraph from 'src/components/graph/PipelineGraph';
import LogViewer from 'src/components/LogViewer';
import {
  argoArtifactUrl,
  argoRequest,
  hasArgoMetadata,
  argoGraph,
  getArgoLogs,
  getArgoMetadata,
  getArgoWorkflow,
  listArgoWorkflows,
  type ArgoWorkflow,
  type ArgoNode,
} from 'src/lib/Argo';
import type { LayerFlowElement } from 'src/lib/v2/FlowTypes';
import { workflowAction } from 'src/lib/ArgoResources';

const errorMessage = (error: unknown) => (error instanceof Error ? error.message : String(error));

/** First native run-details slice; no KFP API providers are mounted here. */
export default function ArgoRunDetails({
  namespace,
  experiment = '',
  experimentNames = {},
  onClearExperiment,
}: {
  namespace: string;
  experiment?: string;
  experimentNames?: Record<string, string>;
  onClearExperiment?: () => void;
}) {
  const [selected, setSelected] = useState<ArgoWorkflow>();
  const [compared, setCompared] = useState<ArgoWorkflow[]>([]);
  const runs = useQuery({
    queryKey: ['argo', namespace, 'runs'],
    queryFn: ({ signal }) => listArgoWorkflows(namespace, signal),
    refetchInterval: 5000,
    retry: false,
  });
  const current = runs.data?.find((run) => run.metadata.uid === selected?.metadata.uid) || selected;
  const visibleRuns = (runs.data || []).filter(
    (run) =>
      !experiment || run.metadata.labels?.['pipelines.kubeflow.org/experiment'] === experiment,
  );
  return (
    <Box sx={{ p: 3, width: '100%', overflow: 'auto' }}>
      <Typography variant='h4'>Kubeflow Pipelines · Argo prototype</Typography>
      <Typography sx={{ my: 1 }}>
        Namespace: {namespace}. Live and archived runs (up to 100 each); refreshed every 5 seconds.
      </Typography>
      <Alert severity='info'>
        Native Argo run details. Workflows are identified by UID; live and archived results are
        deduplicated.
      </Alert>
      {experiment && (
        <Typography>
          Experiment: {experimentNames[experiment] || experiment}{' '}
          <Button onClick={onClearExperiment}>All experiments</Button>
        </Typography>
      )}
      {runs.error && <Alert severity='error'>{errorMessage(runs.error)}</Alert>}
      {runs.isPending && <Typography role='status'>Loading runs…</Typography>}
      <Stack direction='row' spacing={2} sx={{ mt: 2, alignItems: 'flex-start' }}>
        <Paper component='aside' aria-label='Runs' variant='outlined' sx={{ p: 2, minWidth: 280 }}>
          <Typography variant='h6'>Runs</Typography>
          {runs.data && visibleRuns.length === 0 && (
            <Typography>No labeled workflows found.</Typography>
          )}
          {visibleRuns.map((run) => (
            <Box key={run.metadata.uid} sx={{ mb: 1 }}>
              <Button
                onClick={() => setSelected(run)}
                aria-pressed={selected?.metadata.uid === run.metadata.uid}
                variant={selected?.metadata.uid === run.metadata.uid ? 'outlined' : 'text'}
              >
                {run.metadata.name}
              </Button>
              <Typography variant='body2'>
                {run.status?.phase || 'Pending'} {run.archived ? '(archived)' : ''} · Experiment:{' '}
                {experimentNames[
                  run.metadata.labels?.['pipelines.kubeflow.org/experiment'] || ''
                ] ||
                  run.metadata.labels?.['pipelines.kubeflow.org/experiment'] ||
                  'none'}
              </Typography>
              <FormControlLabel
                label='Compare'
                control={
                  <Checkbox
                    checked={compared.some((item) => item.metadata.uid === run.metadata.uid)}
                    disabled={
                      compared.length >= 5 &&
                      !compared.some((item) => item.metadata.uid === run.metadata.uid)
                    }
                    onChange={(_, checked) =>
                      setCompared((items) =>
                        checked
                          ? [...items, run]
                          : items.filter((item) => item.metadata.uid !== run.metadata.uid),
                      )
                    }
                  />
                }
              />
            </Box>
          ))}
        </Paper>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          {compared.length > 0 && <Comparison runs={compared} />}
          {current ? (
            <Run key={current.metadata.uid} run={current} />
          ) : (
            <Typography>Select a run to inspect its Argo graph.</Typography>
          )}
        </Box>
      </Stack>
    </Box>
  );
}

function Run({ run }: { run: ArgoWorkflow }) {
  const client = useQueryClient();
  const action = useMutation({
    mutationFn: (name: 'resume' | 'stop') => workflowAction(run, name),
    onSuccess: () => client.invalidateQueries({ queryKey: ['argo', run.metadata.uid] }),
  });
  const [selection, setSelection] = useState<LayerFlowElement | null>(null);
  const [showLogs, setShowLogs] = useState(false);
  const workflow = useQuery({
    queryKey: ['argo', run.metadata.uid, !!run.archived],
    queryFn: ({ signal }) => getArgoWorkflow(run, signal),
    refetchInterval: 5000,
    retry: false,
  });
  const graph = useMemo(() => argoGraph(workflow.data || run), [workflow.data, run]);
  const node = selection && workflow.data?.status?.nodes?.[selection.element.id];
  const logs = useQuery({
    queryKey: ['argo', run.metadata.uid, 'logs', !!run.archived],
    queryFn: ({ signal }) => getArgoLogs(run, signal),
    enabled: showLogs,
    retry: false,
  });
  return (
    <>
      <Typography variant='h5'>{run.metadata.name}</Typography>
      <Typography>Run UID: {run.metadata.uid}</Typography>
      <Chip sx={{ my: 1 }} label={workflow.data?.status?.phase || 'Pending'} />
      {!run.archived && (
        <>
          <Button disabled={action.isPending} onClick={() => action.mutate('resume')}>
            Resume suspended tasks
          </Button>
          <Button
            disabled={action.isPending}
            onClick={() => {
              if (window.confirm('Stop this run?')) action.mutate('stop');
            }}
          >
            Stop run
          </Button>
        </>
      )}
      {action.error && <Alert severity='error'>{action.error.message}</Alert>}
      <Typography>
        Parameters: {JSON.stringify(workflow.data?.spec.arguments?.parameters || [])}
      </Typography>
      {workflow.error && <Alert severity='error'>{errorMessage(workflow.error)}</Alert>}
      {workflow.data?.status?.message && (
        <Alert severity='warning'>{workflow.data.status.message}</Alert>
      )}
      <Box
        sx={{
          height: 400,
          display: 'flex',
          flexDirection: 'column',
          position: 'relative',
          border: '1px solid #ddd',
          my: 2,
        }}
      >
        <PipelineGraph
          elements={graph}
          selection={selection}
          onSelectionChange={setSelection}
          graphId={run.metadata.uid}
        />
      </Box>
      {node && <Task key={node.id} run={workflow.data!} node={node} />}
      <Button onClick={() => setShowLogs((value) => !value)}>
        {showLogs ? 'Hide logs' : 'Show run logs'}
      </Button>
      {showLogs && (
        <>
          <Typography>
            Last 200 lines per task pod. Cached tasks do not create a new pod.
          </Typography>
          <Button onClick={() => void logs.refetch()}>Refresh logs</Button>
          {logs.error && <Alert severity='error'>{errorMessage(logs.error)}</Alert>}
          {logs.isPending && <Typography role='status'>Loading logs…</Typography>}
          <Box sx={{ height: 280 }}>
            <LogViewer logLines={logs.data || []} />
          </Box>
        </>
      )}
    </>
  );
}

function Task({ run, node }: { run: ArgoWorkflow; node: ArgoNode }) {
  const hasMetadata = hasArgoMetadata(node);
  const [downloadError, setDownloadError] = useState('');
  async function download(port: string) {
    try {
      const response = await argoRequest(argoArtifactUrl(run, node, port));
      const url = URL.createObjectURL(await response.blob());
      const link = document.createElement('a');
      link.href = url;
      link.download = port;
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 0);
      setDownloadError('');
    } catch (error) {
      setDownloadError(errorMessage(error));
    }
  }
  const metadata = useQuery({
    queryKey: ['argo', run.metadata.uid, node.id, 'metadata', !!run.archived],
    queryFn: ({ signal }) => getArgoMetadata(run, node, signal),
    enabled: !!hasMetadata,
    retry: false,
  });
  return (
    <Paper variant='outlined' sx={{ p: 2, my: 2 }}>
      <Typography variant='h6'>
        {node.displayName} · {node.phase}
      </Typography>
      {node.memoizationStatus?.hit && (
        <Alert severity='success'>
          Reused from Argo memoization. Producer below is the original execution.
        </Alert>
      )}
      {metadata.error && <Alert severity='error'>{errorMessage(metadata.error)}</Alert>}
      {downloadError && <Alert severity='error'>{downloadError}</Alert>}
      {!hasMetadata && <Typography>No declared task metadata available.</Typography>}
      {metadata.data && (
        <>
          <Typography>
            Producer: {metadata.data.producer.run} / {metadata.data.producer.task}
          </Typography>
          <Typography variant='h6'>Typed artifacts</Typography>
          {metadata.data.artifacts.map((artifact) => (
            <Box key={artifact.port} sx={{ my: 1 }}>
              <a
                href={argoArtifactUrl(run, node, artifact.port)}
                onClick={(event) => {
                  event.preventDefault();
                  void download(artifact.port);
                }}
              >
                {artifact.port}
              </a>{' '}
              · {artifact.type}
              <Typography sx={{ overflowWrap: 'anywhere' }}>{artifact.uri}</Typography>
              <pre style={{ whiteSpace: 'pre-wrap' }}>
                {JSON.stringify(artifact.metadata, null, 2)}
              </pre>
            </Box>
          ))}
          <Typography variant='h6'>Input lineage</Typography>
          {metadata.data.inputs.map((input) => (
            <Typography key={input.port} sx={{ overflowWrap: 'anywhere' }}>
              {input.port}: {input.uri} — producer {input.producer.run} / {input.producer.task}
            </Typography>
          ))}
          <Typography>Metrics: {JSON.stringify(metadata.data.metrics)}</Typography>
        </>
      )}
    </Paper>
  );
}

function Comparison({ runs }: { runs: ArgoWorkflow[] }) {
  const results = useQueries({
    queries: runs.map((run) => ({
      queryKey: ['argo', run.metadata.uid, 'comparison', !!run.archived],
      queryFn: async ({ signal }: { signal: AbortSignal }) => {
        const workflow = await getArgoWorkflow(run, signal);
        const tasks = Object.values(workflow.status?.nodes || {}).filter(hasArgoMetadata);
        const metadata = await Promise.all(
          tasks.map((node) => getArgoMetadata(workflow, node, signal)),
        );
        return metadata.flatMap((data, index) =>
          Object.entries(data.metrics).map(([name, value]) => ({
            name: `${tasks[index].displayName}.${name}`,
            value,
          })),
        );
      },
      retry: false,
    })),
  });
  return (
    <Paper variant='outlined' sx={{ p: 2, mb: 2 }}>
      <Typography variant='h6'>Selected-run comparison</Typography>
      <table style={{ width: '100%', textAlign: 'left' }}>
        <thead>
          <tr>
            <th>Run</th>
            <th>Metrics</th>
          </tr>
        </thead>
        <tbody>
          {runs.map((run, index) => (
            <tr key={run.metadata.uid}>
              <td>{run.metadata.name}</td>
              <td>
                {results[index].error
                  ? errorMessage(results[index].error)
                  : results[index].isPending
                    ? 'Loading…'
                    : results[index].data
                        ?.map((metric) => `${metric.name}: ${metric.value}`)
                        .join(', ') || 'No metrics'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Paper>
  );
}
