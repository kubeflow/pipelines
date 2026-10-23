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

import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import {
  ArtifactContentPreview as ArtifactContentPreviewData,
  ARTIFACT_PREVIEW_METADATA_KEY,
  DatasetPreview,
  getArtifactPreviewDescriptor,
  ModelPreview,
  ModelPreviewValue,
  parseArtifactContentPreview,
  PreviewKind,
  PreviewScalar,
} from 'src/lib/v2/ArtifactContentPreview';
import WorkflowParser from 'src/lib/WorkflowParser';
import { Artifact } from 'src/third_party/mlmd';

const PREVIEW_ARTIFACT_KINDS: { [typeName: string]: PreviewKind } = {
  'system.Dataset': 'dataset',
  'system.Model': 'model',
};

/** Returns the preview kind for MLMD artifact types that can carry a content preview. */
export function getArtifactContentPreviewKind(
  artifactTypeName: string | undefined,
): PreviewKind | undefined {
  return artifactTypeName ? PREVIEW_ARTIFACT_KINDS[artifactTypeName] : undefined;
}

/**
 * Returns the artifact's preview declaration as producer metadata.
 *
 * The launcher stores dictionary metadata as an MLMD struct wrapped under a "struct" key.
 */
export function getMlmdArtifactPreviewMetadata(
  artifact: Artifact,
): Record<string, unknown> | undefined {
  const value = artifact.getCustomPropertiesMap().get(ARTIFACT_PREVIEW_METADATA_KEY);
  if (!value) {
    return undefined;
  }
  const stored = value.getStructValue()?.toJavaScript();
  const declaration =
    stored && Object.keys(stored).length === 1 && 'struct' in stored ? stored.struct : stored;
  return { [ARTIFACT_PREVIEW_METADATA_KEY]: declaration ?? null };
}

/** Returns whether a Dataset or Model artifact declares a producer-generated preview. */
export function hasArtifactContentPreview(
  artifact: Artifact,
  artifactTypeName: string | undefined,
): boolean {
  return (
    !!getArtifactContentPreviewKind(artifactTypeName) && !!getMlmdArtifactPreviewMetadata(artifact)
  );
}

interface ArtifactContentPreviewProps {
  artifact: Artifact;
  artifactTypeName: string | undefined;
  namespace?: string;
  /** Non-default object store session for the artifact, from its store_session_info property. */
  providerInfo?: string;
}

/** Reads and renders only the bounded _kfp_preview.json beneath one Dataset or Model artifact. */
export function ArtifactContentPreview({
  artifact,
  artifactTypeName,
  namespace,
  providerInfo,
}: ArtifactContentPreviewProps) {
  const kind = getArtifactContentPreviewKind(artifactTypeName);
  const metadata = getMlmdArtifactPreviewMetadata(artifact);
  const uri = artifact.getUri();
  let descriptor;
  let descriptorError: string | undefined;
  try {
    descriptor = getArtifactPreviewDescriptor(metadata);
  } catch (error) {
    descriptorError = error instanceof Error ? error.message : String(error);
  }
  const { data, error, isFetching, refetch } = useQuery<ArtifactContentPreviewData, Error>({
    queryKey: queryKeys.artifactContentPreview(
      artifact.getId(),
      uri,
      kind,
      namespace,
      providerInfo,
      JSON.stringify(metadata?.[ARTIFACT_PREVIEW_METADATA_KEY]),
    ),
    queryFn: async ({ signal }) => {
      if (!kind) {
        throw new Error('Only Dataset and Model artifacts can be previewed.');
      }
      const content = await Apis.getArtifactContentPreview({
        path: WorkflowParser.parseStoragePath(uri),
        namespace,
        providerInfo,
        signal,
      });
      return parseArtifactContentPreview(content, kind);
    },
    enabled: !!descriptor && !!kind && !!uri,
    staleTime: Infinity,
    retry: false,
  });
  if (!kind) {
    return null;
  }
  if (descriptorError) {
    return (
      <Banner
        mode='error'
        message='Invalid artifact preview declaration.'
        additionalInfo={descriptorError}
      />
    );
  }
  if (!descriptor || !uri) {
    return (
      <Banner
        mode='info'
        message='No preview is available for this artifact. Check the pipeline and component preview settings, supported formats, component SDK, and logs.'
      />
    );
  }
  return (
    <>
      {isFetching && <Banner mode='info' message='Artifact preview is loading.' />}
      {error && (
        <>
          <Banner
            mode='error'
            message='Unable to display the artifact preview.'
            additionalInfo={error.message}
          />
          <Button size='small' disabled={isFetching} onClick={() => void refetch()}>
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
        <Banner
          mode='info'
          message='Row values were intentionally omitted by the producer. This is a schema-only preview.'
        />
      ) : rows.length === 0 ? (
        <Banner
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
      <TableContainer style={{ maxWidth: '100%' }}>
        <Table size='small' aria-label='Dataset sample'>
          <TableHead>
            <TableRow>
              {columns.map((column, index) => (
                <TableCell key={index} scope='col' style={{ verticalAlign: 'top' }}>
                  {column.name}
                  <br />
                  <small>{column.dtype}</small>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row, index) => (
              <TableRow key={index}>
                {row.map((cell, columnIndex) => (
                  <TableCell
                    key={columnIndex}
                    style={{
                      minWidth: 100,
                      maxWidth: 300,
                      overflowWrap: 'anywhere',
                      whiteSpace: 'pre-wrap',
                    }}
                  >
                    {displayCell(cell)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
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
        <TableContainer>
          <Table
            size='small'
            aria-label={title}
            sx={{
              tableLayout: 'fixed',
              '& th, & td': {
                overflowWrap: 'anywhere',
                whiteSpace: 'pre-wrap',
                verticalAlign: 'top',
              },
            }}
          >
            <TableHead>
              <TableRow>
                <TableCell scope='col' style={{ width: '40%' }}>
                  Name
                </TableCell>
                <TableCell scope='col'>Recorded value</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {names.map((name) => (
                <TableRow key={name}>
                  <TableCell component='th' scope='row'>
                    {name}
                  </TableCell>
                  <TableCell>{JSON.stringify(entries[name])}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
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
      <DetailsTable
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
