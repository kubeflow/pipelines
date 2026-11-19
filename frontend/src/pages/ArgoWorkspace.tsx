// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Box,
  Button,
  MenuItem,
  Paper,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import * as yaml from 'js-yaml';
import { setArgoToken } from 'src/lib/Argo';
import {
  createResource,
  deleteExperiment,
  listResources,
  setScheduleSuspended,
  submitRun,
  type NativeResource,
} from 'src/lib/ArgoResources';
import ArgoRunDetails from './ArgoRunDetails';

export default function ArgoWorkspace({ namespace }: { namespace: string }) {
  const [token, setToken] = useState('');
  const [connected, setConnected] = useState(false);
  const client = useQueryClient();
  if (!connected)
    return (
      <Box
        component='form'
        onSubmit={(event) => {
          event.preventDefault();
          setArgoToken(token);
          setToken('');
          setConnected(true);
        }}
        sx={{ maxWidth: 650, m: '60px auto', p: 3 }}
      >
        <Typography variant='h4'>Kubeflow Pipelines · Argo</Typography>
        <Typography sx={{ my: 2 }}>
          Connect to namespace {namespace} with a namespace-scoped Kubernetes service-account token.
          It stays in memory and is cleared on reload. Do not use a cluster-admin credential.
        </Typography>
        <TextField
          fullWidth
          label='Kubernetes bearer token'
          type='password'
          autoComplete='off'
          value={token}
          onChange={(event) => setToken(event.target.value)}
        />
        <Button type='submit' sx={{ mt: 2 }}>
          Connect
        </Button>
      </Box>
    );
  return (
    <>
      <Box sx={{ px: 3, pt: 2 }}>
        <Button
          onClick={() => {
            setConnected(false);
            setArgoToken('');
            client.clear();
          }}
        >
          Disconnect
        </Button>
      </Box>
      <Resources namespace={namespace} />
    </>
  );
}

function Resources({ namespace }: { namespace: string }) {
  const [tab, setTab] = useState('runs');
  const [experiment, setExperiment] = useState('');
  const [manifest, setManifest] = useState('');
  const [experimentName, setExperimentName] = useState('');
  const [description, setDescription] = useState('');
  const [versionName, setVersionName] = useState('');
  const [parameters, setParameters] = useState<Record<string, string>>({});
  const [schedule, setSchedule] = useState('0 9 * * *');
  const [result, setResult] = useState('');
  const client = useQueryClient();
  const versions = useQuery({
    queryKey: ['argo', namespace, 'versions'],
    queryFn: ({ signal }) => listResources(namespace, 'workflow-templates', signal),
    refetchInterval: 5000,
    retry: false,
  });
  const experiments = useQuery({
    queryKey: ['argo', namespace, 'experiments'],
    queryFn: ({ signal }) => listResources(namespace, 'experiments', signal),
    refetchInterval: 5000,
    retry: false,
  });
  const schedules = useQuery({
    queryKey: ['argo', namespace, 'schedules'],
    queryFn: ({ signal }) => listResources(namespace, 'cron-workflows', signal),
    refetchInterval: 5000,
    retry: false,
  });
  const mutation = useMutation({
    mutationFn: (operation: () => Promise<any>) => operation(),
    onSuccess: async (value) => {
      setResult(value?.metadata?.name ? `Saved ${value.metadata.name}` : 'Saved');
      await client.invalidateQueries({ queryKey: ['argo'] });
    },
  });
  const version = versions.data?.find((item) => item.metadata.name === versionName);
  function selectVersion(item: NativeResource) {
    setVersionName(item.metadata.name);
    setParameters(
      Object.fromEntries(
        (item.spec?.arguments?.parameters || []).map(
          (parameter: { name: string; value?: string }) => [parameter.name, parameter.value ?? ''],
        ),
      ),
    );
  }
  function createSchedule() {
    if (!version) return;
    const labels = {
      ...version.metadata.labels,
      ...(experiment ? { 'pipelines.kubeflow.org/experiment': experiment } : {}),
    };
    mutation.mutate(() =>
      createResource(namespace, 'cron-workflows', {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'CronWorkflow',
        metadata: { name: 'schedule-' + crypto.randomUUID(), namespace, labels },
        spec: {
          schedules: [schedule],
          suspend: true,
          workflowMetadata: { labels },
          workflowSpec: {
            workflowTemplateRef: { name: version.metadata.name },
            arguments: {
              parameters: Object.entries(parameters).map(([name, value]) => ({ name, value })),
            },
          },
        },
      }),
    );
  }
  return (
    <Box sx={{ width: '100%' }}>
      <Box sx={{ px: 3 }}>
        <Typography variant='h4'>Pipelines on Argo</Typography>
        <Typography>
          Namespace: {namespace}. Kubernetes and Argo are the source of truth.
        </Typography>
        <Tabs value={tab} onChange={(_, value) => setTab(value)} aria-label='Pipeline pages'>
          <Tab value='runs' label='Runs' />
          <Tab value='pipelines' label='Pipelines and versions' />
          <Tab value='experiments' label='Experiments' />
          <Tab value='schedules' label='Recurring runs' />
        </Tabs>
        {mutation.error && <Alert severity='error'>{mutation.error.message}</Alert>}
        {result && (
          <Alert severity='success' onClose={() => setResult('')}>
            {result}
          </Alert>
        )}
        {[versions, experiments, schedules]
          .filter((query) => query.error)
          .map((query, index) => (
            <Alert key={index} severity='error'>
              {query.error!.message}
            </Alert>
          ))}
      </Box>
      {tab === 'runs' && (
        <ArgoRunDetails
          key={namespace + experiment}
          namespace={namespace}
          experiment={experiment}
          experimentNames={Object.fromEntries(
            (experiments.data || []).map((item) => [
              item.metadata.name,
              item.data?.displayName || item.metadata.name,
            ]),
          )}
          onClearExperiment={() => setExperiment('')}
        />
      )}
      {tab === 'pipelines' && (
        <Box sx={{ p: 3 }}>
          <Typography variant='h5'>Pipeline versions · WorkflowTemplates</Typography>
          <Typography>
            Versions are create-only in this UI. Kubernetes itself does not make WorkflowTemplates
            immutable.
          </Typography>
          <Stack direction='row' spacing={3} sx={{ my: 2 }}>
            <Box sx={{ flex: 1 }}>
              {versions.data?.map((item) => (
                <Paper
                  variant='outlined'
                  key={item.metadata.uid || item.metadata.name}
                  sx={{ p: 2, mb: 1 }}
                >
                  <Typography variant='h6'>
                    {item.metadata.annotations?.['pipelines.kubeflow.org/display-name'] ||
                      item.metadata.labels?.['pipelines.kubeflow.org/pipeline']}
                  </Typography>
                  <Typography>{item.metadata.name}</Typography>
                  <Typography>
                    {item.metadata.annotations?.['pipelines.kubeflow.org/description']}
                  </Typography>
                  <Button onClick={() => selectVersion(item)}>Use {item.metadata.name}</Button>
                </Paper>
              ))}
              <TextField
                label='WorkflowTemplate YAML'
                multiline
                minRows={8}
                fullWidth
                value={manifest}
                onChange={(event) => setManifest(event.target.value)}
              />
              <Button
                disabled={mutation.isPending || !manifest}
                onClick={() =>
                  mutation.mutate(async () =>
                    createResource(
                      namespace,
                      'workflow-templates',
                      yaml.load(manifest) as NativeResource,
                    ),
                  )
                }
              >
                Upload version
              </Button>
            </Box>
            <Paper variant='outlined' sx={{ p: 2, flex: 1 }}>
              <Typography variant='h6'>Create a run or schedule</Typography>
              <Typography>{versionName || 'Choose a version first.'}</Typography>
              <TextField
                select
                fullWidth
                label='Experiment'
                value={experiment}
                onChange={(event) => setExperiment(event.target.value)}
                sx={{ my: 2 }}
              >
                <MenuItem value=''>No experiment</MenuItem>
                {experiments.data?.map((item) => (
                  <MenuItem key={item.metadata.name} value={item.metadata.name}>
                    {item.data?.displayName || item.metadata.name}
                  </MenuItem>
                ))}
              </TextField>
              {Object.entries(parameters).map(([name, value]) => (
                <TextField
                  key={name}
                  fullWidth
                  label={name}
                  value={value}
                  onChange={(event) =>
                    setParameters((previous) => ({ ...previous, [name]: event.target.value }))
                  }
                  sx={{ mb: 2 }}
                />
              ))}
              <Button
                disabled={!version || mutation.isPending}
                onClick={() =>
                  version &&
                  mutation.mutate(() => submitRun(namespace, version, parameters, experiment))
                }
              >
                Start run
              </Button>
              <TextField
                fullWidth
                label='Cron schedule'
                value={schedule}
                onChange={(event) => setSchedule(event.target.value)}
                sx={{ my: 2 }}
              />
              <Button disabled={!version || mutation.isPending} onClick={createSchedule}>
                Create suspended schedule
              </Button>
            </Paper>
          </Stack>
        </Box>
      )}
      {tab === 'experiments' && (
        <Box sx={{ p: 3 }}>
          <Typography variant='h5'>Experiments</Typography>
          <Typography>
            Each experiment is a ConfigMap. Deleting it does not delete any run or schedule.
          </Typography>
          <Box
            component='form'
            onSubmit={(event) => {
              event.preventDefault();
              mutation.mutate(() =>
                createResource(namespace, 'experiments', {
                  apiVersion: 'v1',
                  kind: 'ConfigMap',
                  metadata: {
                    name: 'experiment-' + crypto.randomUUID(),
                    namespace,
                    labels: { 'pipelines.kubeflow.org/resource': 'experiment' },
                  },
                  data: { displayName: experimentName, description },
                }),
              );
            }}
            sx={{ my: 2 }}
          >
            <TextField
              required
              label='Experiment name'
              value={experimentName}
              onChange={(event) => setExperimentName(event.target.value)}
            />
            <TextField
              label='Description'
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
            <Button type='submit' disabled={mutation.isPending}>
              Create experiment
            </Button>
          </Box>
          {experiments.data?.map((item) => (
            <Paper variant='outlined' key={item.metadata.uid} sx={{ p: 2, mb: 1 }}>
              <Typography variant='h6'>{item.data?.displayName || item.metadata.name}</Typography>
              <Typography>{item.data?.description}</Typography>
              <Typography>{item.metadata.name}</Typography>
              <Button
                onClick={() => {
                  setExperiment(item.metadata.name);
                  setTab('runs');
                }}
              >
                View runs
              </Button>
              <Button
                disabled={mutation.isPending}
                onClick={() => {
                  if (
                    window.confirm(
                      `Delete experiment ${item.data?.displayName || item.metadata.name}? Runs and schedules will remain.`,
                    )
                  )
                    mutation.mutate(() => deleteExperiment(namespace, item));
                }}
              >
                Delete experiment
              </Button>
            </Paper>
          ))}
        </Box>
      )}
      {tab === 'schedules' && (
        <Box sx={{ p: 3 }}>
          <Typography variant='h5'>Recurring runs · CronWorkflows</Typography>
          {schedules.data?.map((item) => (
            <Paper variant='outlined' key={item.metadata.uid} sx={{ p: 2, my: 1 }}>
              <Typography variant='h6'>{item.metadata.name}</Typography>
              <Typography>
                {(item.spec?.schedules || [item.spec?.schedule]).join(', ')} ·{' '}
                {item.spec?.suspend ? 'Suspended' : 'Active'}
              </Typography>
              <Button
                disabled={mutation.isPending}
                onClick={() =>
                  mutation.mutate(() => setScheduleSuspended(namespace, item, !item.spec?.suspend))
                }
              >
                {item.spec?.suspend ? 'Enable schedule' : 'Suspend schedule'}
              </Button>
            </Paper>
          ))}
        </Box>
      )}
    </Box>
  );
}
