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
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { Apis } from 'src/lib/Apis';
import { Artifact, Value } from 'src/third_party/mlmd';
import { CommonTestWrapper } from 'src/TestWrapper';
import { ArtifactContentPreview, hasArtifactContentPreview } from './ArtifactContentPreview';
import dataset from '../../../mock-backend/data/v2/artifact-previews/dataset.json';
import model from '../../../mock-backend/data/v2/artifact-previews/model.json';

function buildArtifact(
  declaration: unknown = { version: 1, path: 'preview.json' },
  uri = 's3://bucket/private-artifacts/team-a/dataset',
): Artifact {
  const artifact = new Artifact();
  artifact.setId(1);
  artifact.setUri(uri);
  if (declaration !== null) {
    // The launcher stores dictionary metadata wrapped under a "struct" key.
    artifact
      .getCustomPropertiesMap()
      .set(
        '_kfp_preview',
        new Value().setStructValue(Struct.fromJavaScript({ struct: declaration as never })),
      );
  }
  return artifact;
}

function view(
  artifact = buildArtifact(),
  typeName = 'system.Dataset',
  namespace: string | undefined = 'team-a',
) {
  return (
    <CommonTestWrapper>
      <ArtifactContentPreview
        artifact={artifact}
        artifactTypeName={typeName}
        namespace={namespace}
        providerInfo='{"provider":"minio"}'
      />
    </CommonTestWrapper>
  );
}

afterEach(() => vi.restoreAllMocks());

describe('artifact content previews', () => {
  it('reads the preview beneath the artifact and shows dimensions and dtypes', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(dataset));
    const rawRead = vi.spyOn(Apis, 'readFile');
    render(view());
    const table = await screen.findByRole('table', { name: 'Dataset sample' });
    expect(within(table).getAllByRole('row')).toHaveLength(11);
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('0.1');
    expect(screen.getByRole('status')).toHaveTextContent('Showing 10 of 1000 rows; 3 of 3 columns');
    expect(screen.getByText('float64')).toBeVisible();
    expect(read).toHaveBeenCalledWith({
      path: { source: 's3', bucket: 'bucket', key: 'private-artifacts/team-a/dataset' },
      namespace: 'team-a',
      providerInfo: '{"provider":"minio"}',
      signal: expect.anything(),
    });
    expect(rawRead).not.toHaveBeenCalled();
  });

  it('keeps missing and malformed declarations out of the network path', () => {
    const read = vi.spyOn(Apis, 'getArtifactContentPreview');
    const { rerender } = render(view(buildArtifact(null)));
    expect(screen.getByText(/pipeline and component preview settings/)).toBeVisible();
    rerender(view(buildArtifact({ version: 2, path: 'preview.json' })));
    expect(screen.getByText('Invalid artifact preview declaration.')).toBeVisible();
    expect(read).not.toHaveBeenCalled();
  });

  it('only previews Dataset and Model artifacts that declare a preview', () => {
    expect(hasArtifactContentPreview(buildArtifact(), 'system.Dataset')).toBe(true);
    expect(hasArtifactContentPreview(buildArtifact(), 'system.Model')).toBe(true);
    expect(hasArtifactContentPreview(buildArtifact(), 'system.Metrics')).toBe(false);
    expect(hasArtifactContentPreview(buildArtifact(null), 'system.Dataset')).toBe(false);
    const read = vi.spyOn(Apis, 'getArtifactContentPreview');
    const { container } = render(view(buildArtifact(), 'system.Metrics'));
    expect(container).toBeEmptyDOMElement();
    expect(read).not.toHaveBeenCalled();
  });

  it('separates cached previews by namespace', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(dataset));
    const { rerender } = render(view());
    await screen.findByRole('table', { name: 'Dataset sample' });
    rerender(view(buildArtifact(), 'system.Dataset', 'team-b'));
    await waitFor(() =>
      expect(read).toHaveBeenLastCalledWith(expect.objectContaining({ namespace: 'team-b' })),
    );
  });

  it('cancels in-flight reads when the preview unmounts', async () => {
    let requestSignal: AbortSignal | undefined;
    vi.spyOn(Apis, 'getArtifactContentPreview').mockImplementation(({ signal }) => {
      requestSignal = signal;
      return new Promise((_resolve, reject) => {
        signal?.addEventListener('abort', () => reject(new Error('cancelled')), { once: true });
      });
    });
    const { unmount } = render(view());
    await screen.findByText('Artifact preview is loading.');
    unmount();
    expect(requestSignal?.aborted).toBe(true);
  });

  it('shows read errors with an explicit retry and rejects malformed files', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockRejectedValue(new Error('preview.json is missing'));
    render(view());
    await screen.findByText('Unable to display the artifact preview.');
    const attempts = read.mock.calls.length;
    read.mockResolvedValue('{not JSON');
    fireEvent.click(screen.getByRole('button', { name: 'Retry preview' }));
    await waitFor(() => expect(read).toHaveBeenCalledTimes(attempts + 1));
    expect(screen.queryByRole('table')).toBeNull();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Retry preview' })).toBeEnabled(),
    );
    read.mockResolvedValue(JSON.stringify(dataset));
    fireEvent.click(screen.getByRole('button', { name: 'Retry preview' }));
    await screen.findByRole('table', { name: 'Dataset sample' });
  });

  it('renders values as text and distinguishes null and empty strings', async () => {
    vi.spyOn(Apis, 'getArtifactContentPreview').mockResolvedValue(
      JSON.stringify({ ...dataset, rows: [['<script>alert(1)</script>', null, '']] }),
    );
    const { container } = render(view());
    expect(await screen.findByText('<script>alert(1)</script>')).toBeVisible();
    expect(screen.getByText('null')).toBeVisible();
    expect(screen.getByText('""')).toBeVisible();
    expect(container.querySelector('script')).toBeNull();
  });

  it('renders model identity, recorded parameters, and structural facts separately', async () => {
    vi.spyOn(Apis, 'getArtifactContentPreview').mockResolvedValue(JSON.stringify(model));
    render(
      view(
        buildArtifact({ version: 1, path: 'preview.json' }, 's3://bucket/model'),
        'system.Model',
      ),
    );
    expect(await screen.findByRole('region', { name: 'Model preview' })).toBeVisible();
    expect(screen.getByText('xgboost.XGBClassifier')).toBeVisible();
    expect(screen.getByRole('heading', { name: 'Recorded hyperparameters' })).toBeVisible();
    expect(screen.getByRole('heading', { name: 'Structural information' })).toBeVisible();
    const parameters = screen.getByRole('table', { name: 'Recorded hyperparameters' });
    expect(
      within(parameters)
        .getAllByRole('rowheader')
        .map((cell) => cell.textContent),
    ).toEqual(Object.keys(model.hyperparameters).sort());
    const structure = screen.getByRole('table', { name: 'Structural information' });
    expect(within(structure).getAllByRole('row')).toHaveLength(
      Object.keys(model.structure).length + 1,
    );
  });
});
