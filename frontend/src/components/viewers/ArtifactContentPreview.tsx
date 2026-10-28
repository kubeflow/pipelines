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

import { Button } from '../ui/button';
import { InspectionNotice } from '../inspection/InspectionNotice';
import { InspectionFields } from '../inspection/InspectionFields';
import './RichViewers.css';
import './ComparisonViewers.css';
import { useQuery } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { V2Artifact } from 'src/apisv2/run';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import {
  ArtifactContentPreview,
  ArtifactPreviewDescriptor,
  DatasetPreview,
  getArtifactPreviewDescriptor,
  ModelPreview,
  ModelPreviewValue,
  parseArtifactContentPreview,
  PreviewScalar,
} from 'src/lib/v2/ArtifactContentPreview';
import {
  getArtifactContentPreviewKind,
  getArtifactDisplayName,
  getArtifactIdentity,
} from 'src/lib/v2/RuntimeArtifactUtils';

interface PreviewContext {
  namespace?: string;
  active?: boolean;
  sourceFinished?: boolean;
  refreshGeneration?: number;
}

/** Selects one artifact without eagerly downloading previews for the rest of an output group. */
export function ArtifactContentPreviews({
  artifacts,
  active = true,
  ...context
}: PreviewContext & { artifacts: V2Artifact[] }) {
  const labelId = useId();
  const [selectedKey, setSelectedKey] = useState('');
  const entries = [
    ...new Map(
      artifacts.map((artifact, index) => [
        JSON.stringify([
          artifact.namespace || context.namespace,
          getArtifactIdentity(artifact) || index,
        ]),
        artifact,
      ]),
    ).entries(),
  ];
  const selected =
    entries.find(([key]) => key === selectedKey) || (entries.length === 1 ? entries[0] : undefined);
  if (!entries.length) {
    return null;
  }
  return (
    <div className='kfp-visualization-section'>
      {entries.length > 1 && (
        <label className='kfp-visualization-field' htmlFor={labelId}>
          Artifact preview
          <select
            id={labelId}
            className='kfp-visualization-select'
            value={selected?.[0] || ''}
            onChange={(event) => setSelectedKey(event.target.value)}
          >
            <option value=''>Choose an artifact</option>
            {entries.map(([key, artifact], index) => (
              <option key={key} value={key}>
                {getArtifactDisplayName(artifact, undefined, index, artifacts)}
              </option>
            ))}
          </select>
        </label>
      )}
      {active && selected && (
        <SelectedArtifactPreview key={selected[0]} artifact={selected[1]} {...context} />
      )}
    </div>
  );
}

function SelectedArtifactPreview({
  artifact,
  namespace,
  sourceFinished,
  refreshGeneration = 0,
}: PreviewContext & { artifact: V2Artifact }) {
  const kind = getArtifactContentPreviewKind(artifact);
  const effectiveNamespace = artifact.namespace || namespace;
  let descriptor: ArtifactPreviewDescriptor | undefined;
  let descriptorError: string | undefined;
  try {
    descriptor = getArtifactPreviewDescriptor(artifact.metadata);
  } catch (error) {
    descriptorError = error instanceof Error ? error.message : String(error);
  }
  const { data, error, isFetching, refetch } = useQuery<ArtifactContentPreview, Error>({
    queryKey: queryKeys.artifactContentPreview(
      artifact.artifact_id,
      artifact.uri,
      kind,
      effectiveNamespace,
      JSON.stringify(artifact.metadata?._kfp_preview),
      sourceFinished,
      refreshGeneration,
    ),
    queryFn: async ({ signal }) => {
      if (!artifact.artifact_id || !kind) {
        throw new Error(
          'The artifact must be registered as a Dataset or Model before it can be previewed.',
        );
      }
      const content = await Apis.getArtifactContentPreview(
        artifact.artifact_id,
        effectiveNamespace,
        signal,
      );
      return parseArtifactContentPreview(content, kind);
    },
    enabled: !!descriptor && !!kind && !!artifact.artifact_id,
    staleTime: Infinity,
    retry: false,
  });
  if (descriptorError) {
    return (
      <InspectionNotice
        mode='error'
        message='Invalid artifact preview declaration.'
        additionalInfo={descriptorError}
      />
    );
  }
  if (!descriptor) {
    return (
      <InspectionNotice
        mode='info'
        message='No preview is available for this artifact. Check the pipeline and component preview settings, supported formats, component SDK, and logs.'
      />
    );
  }
  if (!artifact.artifact_id) {
    return (
      <InspectionNotice
        mode='info'
        message='The artifact preview will be available after the artifact is registered.'
      />
    );
  }
  return (
    <>
      {isFetching && <InspectionNotice mode='info' message='Artifact preview is loading.' />}
      {error && (
        <>
          <InspectionNotice
            mode='error'
            message='Unable to display the artifact preview.'
            additionalInfo={error.message}
          />
          <Button
            size='sm'
            variant='secondary'
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            Retry preview
          </Button>
        </>
      )}
      {data &&
        (data.kind === 'dataset' ? (
          <DatasetPreviewTable preview={data} />
        ) : (
          <ModelPreviewSummary preview={data} />
        ))}
    </>
  );
}

function displayCell(value: PreviewScalar): string {
  return value === '' ? '""' : String(value);
}

export function DatasetPreviewTable({ preview }: { preview: DatasetPreview }) {
  const { columns, rows, sample, total_rows, total_columns } = preview;
  return (
    <section aria-label='Dataset preview'>
      <h3>Dataset preview{preview.format ? ` (${preview.format})` : ''}</h3>
      <p role='status'>
        Showing {rows.length} of {total_rows ?? 'unknown'} rows; {columns.length} of{' '}
        {total_columns ?? 'unknown'} columns. Sample: {sample.method}.
      </p>
      {sample.rows_omitted ? (
        <InspectionNotice
          mode='info'
          message='Row values were intentionally omitted by the producer. This is a schema-only preview.'
        />
      ) : rows.length === 0 ? (
        <InspectionNotice
          mode='info'
          message={total_rows === 0 ? 'Dataset is empty.' : 'No sample rows were recorded.'}
        />
      ) : null}
      {(sample.rows_truncated ||
        sample.columns_truncated ||
        sample.values_truncated ||
        sample.redacted) && (
        <p>
          {sample.rows_truncated && 'Only sampled rows are shown. '}
          {sample.columns_truncated && 'Some columns are not included. '}
          {sample.values_truncated && 'Some values were shortened. '}
          {sample.redacted && 'The producer redacted sample values.'}
        </p>
      )}
      <div className='kfp-viewer-table kfp-viewer-table-scroll'>
        <table aria-label='Dataset sample'>
          <thead>
            <tr>
              {columns.map((column, index) => (
                <th key={index} scope='col' style={{ verticalAlign: 'top' }}>
                  {column.name}
                  <br />
                  <small>{column.dtype}</small>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr className='kfp-viewer-table-data' key={index}>
                {row.map((cell, columnIndex) => (
                  <td
                    key={columnIndex}
                    style={{
                      minWidth: 100,
                      maxWidth: 300,
                      overflowWrap: 'anywhere',
                      whiteSpace: 'pre-wrap',
                    }}
                  >
                    {displayCell(cell)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function ModelEntries({
  title,
  entries,
}: {
  title: string;
  entries: Record<string, ModelPreviewValue>;
}) {
  const names = Object.keys(entries).sort();
  return (
    <>
      <h4>{title}</h4>
      {names.length ? (
        <div className='kfp-viewer-table kfp-viewer-table-scroll'>
          <table aria-label={title} style={{ tableLayout: 'fixed' }}>
            <thead>
              <tr>
                <th scope='col' style={{ width: '40%' }}>
                  Name
                </th>
                <th scope='col'>Recorded value</th>
              </tr>
            </thead>
            <tbody>
              {names.map((name) => (
                <tr className='kfp-viewer-table-data' key={name}>
                  <th scope='row' style={{ overflowWrap: 'anywhere', whiteSpace: 'pre-wrap' }}>
                    {name}
                  </th>
                  <td style={{ overflowWrap: 'anywhere', whiteSpace: 'pre-wrap' }}>
                    {JSON.stringify(entries[name])}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p>Not recorded.</p>
      )}
    </>
  );
}

export function ModelPreviewSummary({ preview }: { preview: ModelPreview }) {
  return (
    <section aria-label='Model preview'>
      <h3>Model preview</h3>
      <InspectionFields
        fields={[
          ['Framework', preview.framework?.name || 'Not recorded'],
          ['Framework version', preview.framework?.version || 'Not recorded'],
          ['Estimator class', preview.estimator_class || 'Not recorded'],
        ]}
      />
      <p>Producer-recorded configuration and structural facts, not learned weights.</p>
      <ModelEntries title='Recorded hyperparameters' entries={preview.hyperparameters} />
      <ModelEntries title='Structural information' entries={preview.structure} />
      {preview.notes.map((note, index) => (
        <p key={index}>{note}</p>
      ))}
    </section>
  );
}
