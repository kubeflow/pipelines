// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as React from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import {
  downloadMetadata,
  exportMetadata,
  importMetadata,
  MAX_TRANSFER_BYTES,
  TransferResult,
} from 'src/lib/MetadataTransfer';
import { PageProps } from './Page';

export function MetadataTransferForm({
  namespace,
  disabled = false,
}: {
  namespace: string;
  disabled?: boolean;
}): React.JSX.Element {
  const [after, setAfter] = React.useState('');
  const [before, setBefore] = React.useState('');
  const [file, setFile] = React.useState<File>();
  const [prefix, setPrefix] = React.useState('imported-');
  const [preview, setPreview] = React.useState<TransferResult>();
  const [result, setResult] = React.useState<TransferResult>();
  const [busy, setBusy] = React.useState<'export' | 'validate' | 'import'>();
  const [error, setError] = React.useState('');
  const [downloaded, setDownloaded] = React.useState(false);
  const request = React.useRef<AbortController | null>(null);
  // External synchronization: cancel browser requests when the namespace/page is left.
  React.useEffect(() => () => request.current?.abort(), []);
  const locked = disabled || !!busy;

  function begin(operation: 'export' | 'validate' | 'import'): AbortController | undefined {
    if (request.current || disabled) return undefined;
    const controller = new AbortController();
    request.current = controller;
    setBusy(operation);
    setError('');
    return controller;
  }
  function fail(cause: unknown, controller: AbortController): void {
    if (!controller.signal.aborted)
      setError(cause instanceof Error ? cause.message : 'The request failed. Try again.');
  }
  function finish(controller: AbortController): void {
    if (!controller.signal.aborted) setBusy(undefined);
    if (request.current === controller) request.current = null;
  }
  async function exportArchive(): Promise<void> {
    const controller = begin('export');
    if (!controller) return;
    setDownloaded(false);
    try {
      // datetime-local supplies no timezone; these fields always represent UTC.
      const completed_after = after ? Date.parse(`${after}Z`) / 1000 : undefined;
      const completed_before = before ? Date.parse(`${before}Z`) / 1000 : undefined;
      if (
        (completed_after !== undefined && !Number.isFinite(completed_after)) ||
        (completed_before !== undefined && !Number.isFinite(completed_before)) ||
        (completed_after !== undefined &&
          completed_before !== undefined &&
          completed_after >= completed_before)
      ) {
        throw new Error('Choose an end time later than the start time.');
      }
      const archive = await exportMetadata(
        namespace,
        { completed_after, completed_before },
        controller.signal,
      );
      if (!controller.signal.aborted) {
        downloadMetadata(archive);
        setDownloaded(true);
      }
    } catch (cause) {
      fail(cause, controller);
    } finally {
      finish(controller);
    }
  }
  async function importArchive(dryRun: boolean): Promise<void> {
    if (!file || (!dryRun && !preview)) return;
    const controller = begin(dryRun ? 'validate' : 'import');
    if (!controller) return;
    setResult(undefined);
    if (dryRun) setPreview(undefined);
    try {
      const response = await importMetadata(namespace, file, prefix, dryRun, controller.signal);
      if (!controller.signal.aborted) {
        if (dryRun) setPreview(response);
        else {
          setResult(response);
          setPreview(undefined);
        }
      }
    } catch (cause) {
      if (!controller.signal.aborted) setPreview(undefined);
      fail(cause, controller);
    } finally {
      finish(controller);
    }
  }
  function invalidate(): void {
    setPreview(undefined);
    setResult(undefined);
    setError('');
  }
  const summary = result || preview;
  return (
    <Stack spacing={3} sx={{ maxWidth: 960, p: 3 }}>
      <Typography variant='body1'>
        Move pipeline metadata between installations. Namespace:{' '}
        <strong>{namespace || 'Installation default'}</strong>
      </Typography>
      <Alert severity='info'>
        Archives include experiments, pipeline definitions and versions, completed run history, and
        recurring runs. Artifact files and logs are not copied; their existing storage locations
        must remain accessible. Imported recurring runs have scheduling and catchup disabled. You
        can edit them after import.
      </Alert>
      {disabled && (
        <Alert severity='warning'>Select a namespace before exporting or importing.</Alert>
      )}
      {error && <Alert severity='error'>{error}</Alert>}
      <Paper variant='outlined' sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography component='h2' variant='h6'>
            Export metadata
          </Typography>
          <Typography variant='body2'>
            Export the full experiment and pipeline catalog, including empty experiments, and all
            recurring runs. Active runs are excluded. Optional times limit completed run history
            only. Narrow the time range to export busy periods in smaller batches.
          </Typography>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
            <TextField
              label='Completed from (UTC)'
              type='datetime-local'
              inputProps={{ step: 1 }}
              sx={{ minWidth: 280 }}
              value={after}
              disabled={locked}
              InputLabelProps={{ shrink: true }}
              onChange={(event) => setAfter(event.target.value)}
            />
            <TextField
              label='Completed before (UTC)'
              type='datetime-local'
              inputProps={{ step: 1 }}
              sx={{ minWidth: 280 }}
              value={before}
              disabled={locked}
              InputLabelProps={{ shrink: true }}
              onChange={(event) => setBefore(event.target.value)}
            />
          </Stack>
          <Box>
            <Button variant='contained' disabled={locked} onClick={() => void exportArchive()}>
              {busy === 'export' ? 'Exporting…' : 'Download archive'}
            </Button>
          </Box>
          {downloaded && <Alert severity='success'>Archive download started.</Alert>}
        </Stack>
      </Paper>
      <Paper variant='outlined' sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography component='h2' variant='h6'>
            Import metadata
          </Typography>
          <Typography variant='body2'>
            Choose an archive from the same Pipelines generation and namespace. Validate it before
            importing into this installation. Existing records are never overwritten.
          </Typography>
          <Box>
            <Typography component='label' htmlFor='metadata-archive' variant='body2'>
              Metadata archive (JSON, up to 256 MiB)
            </Typography>
            <Box
              component='input'
              id='metadata-archive'
              type='file'
              accept='.json,application/json'
              disabled={locked}
              sx={{ display: 'block', mt: 1 }}
              onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
                invalidate();
                const selected = event.target.files?.[0];
                if (selected && selected.size > MAX_TRANSFER_BYTES) {
                  setFile(undefined);
                  setError('Choose an archive no larger than 256 MiB.');
                } else setFile(selected);
              }}
            />
          </Box>
          <TextField
            label='Imported name prefix'
            value={prefix}
            disabled={locked}
            helperText='Applied to imported experiment and pipeline names to avoid name conflicts.'
            onChange={(event) => {
              setPrefix(event.target.value);
              invalidate();
            }}
          />
          <Stack direction='row' spacing={2}>
            <Button
              variant='outlined'
              disabled={locked || !file}
              onClick={() => void importArchive(true)}
            >
              {busy === 'validate' ? 'Validating…' : 'Validate archive'}
            </Button>
            <Button
              variant='contained'
              disabled={locked || !preview}
              onClick={() => void importArchive(false)}
            >
              {busy === 'import' ? 'Importing…' : 'Import metadata'}
            </Button>
          </Stack>
          {summary && (
            <Box role='status'>
              <Typography variant='subtitle1'>
                {result ? 'Import complete' : 'Archive validated. Ready to import.'}
              </Typography>
              <Box
                component='dl'
                sx={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 1fr',
                  gap: 1,
                  '& dt, & dd': { m: 0 },
                }}
              >
                {Object.entries({
                  Experiments: summary.counts.experiments,
                  Pipelines: summary.counts.pipelines,
                  'Pipeline versions': summary.counts.pipeline_versions,
                  'Completed runs': summary.counts.runs,
                  'Recurring runs': summary.counts.schedules,
                }).map(([label, count]) => (
                  <React.Fragment key={label}>
                    <Typography component='dt'>{label}</Typography>
                    <Typography component='dd'>{count}</Typography>
                  </React.Fragment>
                ))}
              </Box>
              <Typography variant='body2'>
                {summary.imported} {result ? 'imported' : 'to import'}; {summary.skipped} already
                imported.
              </Typography>
              {summary.warnings.map((warning, index) => (
                <Alert severity='warning' key={`${index}-${warning}`} sx={{ mt: 1 }}>
                  {warning}
                </Alert>
              ))}
            </Box>
          )}
        </Stack>
      </Paper>
    </Stack>
  );
}

export default function MetadataTransferPage({ updateToolbar }: PageProps): React.JSX.Element {
  const namespace = React.useContext(NamespaceContext) || '';
  const buildInfo = React.useContext(BuildInfoContext);
  // External synchronization: the toolbar is owned by the router outside this page.
  React.useEffect(() => {
    updateToolbar({ pageTitle: 'Export / Import', actions: {}, breadcrumbs: [] });
  }, [updateToolbar]);
  return (
    <MetadataTransferForm
      key={namespace}
      namespace={namespace}
      disabled={!!buildInfo?.apiServerMultiUser && !namespace}
    />
  );
}
