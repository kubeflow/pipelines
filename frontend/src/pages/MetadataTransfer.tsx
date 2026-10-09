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
import { Alert } from 'src/components/ui/alert';
import { Button } from 'src/components/ui/button';
import { TextField } from 'src/components/ui/text-field';
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
    <div
      className='kfp-inspection-scroll'
      style={{ maxWidth: 960, padding: 24, display: 'grid', gap: 24 }}
    >
      <p>
        Move pipeline metadata between installations. Namespace:{' '}
        <strong>{namespace || 'Installation default'}</strong>
      </p>
      <Alert variant='info'>
        Archives include experiments, pipeline definitions and versions, completed run history, and
        recurring runs. Artifact files and logs are not copied; their existing storage locations
        must remain accessible. Imported recurring runs have scheduling and catchup disabled. You
        can edit them after import.
      </Alert>
      {disabled && (
        <Alert variant='warning'>Select a namespace before exporting or importing.</Alert>
      )}
      {error && <Alert variant='error'>{error}</Alert>}
      <section className='kfp-inspection-details-card' style={{ padding: 24 }}>
        <div style={{ display: 'grid', gap: 16 }}>
          <h2>Export metadata</h2>
          <p>
            Export the full experiment and pipeline catalog, including empty experiments, and all
            recurring runs. Active runs are excluded. Optional times limit completed run history
            only. Narrow the time range to export busy periods in smaller batches.
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
            <TextField
              label='Completed from (UTC)'
              type='datetime-local'
              step={1}
              style={{ minWidth: 280 }}
              value={after}
              disabled={locked}
              onChange={(event) => setAfter(event.target.value)}
            />
            <TextField
              label='Completed before (UTC)'
              type='datetime-local'
              step={1}
              style={{ minWidth: 280 }}
              value={before}
              disabled={locked}
              onChange={(event) => setBefore(event.target.value)}
            />
          </div>
          <div>
            <Button variant='default' disabled={locked} onClick={() => void exportArchive()}>
              {busy === 'export' ? 'Exporting…' : 'Download archive'}
            </Button>
          </div>
          {downloaded && <Alert variant='info'>Archive download started.</Alert>}
        </div>
      </section>
      <section className='kfp-inspection-details-card' style={{ padding: 24 }}>
        <div style={{ display: 'grid', gap: 16 }}>
          <h2>Import metadata</h2>
          <p>
            Choose an archive from the same Pipelines generation and namespace. Validate it before
            importing into this installation. Existing records are never overwritten.
          </p>
          <div>
            <label htmlFor='metadata-archive'>Metadata archive (JSON, up to 256 MiB)</label>
            <input
              id='metadata-archive'
              type='file'
              accept='.json,application/json'
              disabled={locked}
              style={{ display: 'block', marginTop: 8 }}
              onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
                invalidate();
                const selected = event.target.files?.[0];
                if (selected && selected.size > MAX_TRANSFER_BYTES) {
                  setFile(undefined);
                  setError('Choose an archive no larger than 256 MiB.');
                } else setFile(selected);
              }}
            />
          </div>
          <TextField
            label='Imported name prefix'
            value={prefix}
            disabled={locked}
            hint='Applied to imported experiment and pipeline names to avoid name conflicts.'
            onChange={(event) => {
              setPrefix(event.target.value);
              invalidate();
            }}
          />
          <div style={{ display: 'flex', gap: 16 }}>
            <Button
              variant='secondary'
              disabled={locked || !file}
              onClick={() => void importArchive(true)}
            >
              {busy === 'validate' ? 'Validating…' : 'Validate archive'}
            </Button>
            <Button
              variant='default'
              disabled={locked || !preview}
              onClick={() => void importArchive(false)}
            >
              {busy === 'import' ? 'Importing…' : 'Import metadata'}
            </Button>
          </div>
          {summary && (
            <div role='status' aria-label='Import result'>
              <p>{result ? 'Import complete' : 'Archive validated. Ready to import.'}</p>
              <dl style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                {Object.entries({
                  Experiments: summary.counts.experiments,
                  Pipelines: summary.counts.pipelines,
                  'Pipeline versions': summary.counts.pipeline_versions,
                  'Completed runs': summary.counts.runs,
                  'Recurring runs': summary.counts.schedules,
                }).map(([label, count]) => (
                  <React.Fragment key={label}>
                    <dt>{label}</dt>
                    <dd>{count}</dd>
                  </React.Fragment>
                ))}
              </dl>
              <p>
                {summary.imported} {result ? 'imported' : 'to import'}; {summary.skipped} already
                imported.
              </p>
              {summary.warnings.map((warning, index) => (
                <Alert variant='warning' key={`${index}-${warning}`} className='mt-2'>
                  {warning}
                </Alert>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
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
