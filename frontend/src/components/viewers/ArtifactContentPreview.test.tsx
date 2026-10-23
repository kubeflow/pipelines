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
import { ArtifactArtifactType, V2Artifact } from 'src/apisv2/run';
import { Apis } from 'src/lib/Apis';
import { CommonTestWrapper } from 'src/TestWrapper';
import { ArtifactContentPreviews } from './ArtifactContentPreview';
import type { MockInstance } from 'vitest';
import dataset from '../../../mock-backend/data/v2/artifact-previews/dataset.json';
import model from '../../../mock-backend/data/v2/artifact-previews/model.json';

const artifact: V2Artifact = {
  artifact_id: 'dataset-1',
  name: 'training data',
  type: ArtifactArtifactType.Dataset,
  namespace: 'team-a',
  uri: 's3://bucket/private-artifacts/team-a/dataset',
  metadata: { _kfp_preview: { version: 1, path: '_kfp_preview.json' } },
};

function view(artifacts = [artifact], active = true, sourceFinished = false) {
  return (
    <CommonTestWrapper>
      <ArtifactContentPreviews
        artifacts={artifacts}
        active={active}
        sourceFinished={sourceFinished}
      />
    </CommonTestWrapper>
  );
}

afterEach(() => vi.restoreAllMocks());

function completedRequests(read: MockInstance<typeof Apis.getArtifactContentPreview>) {
  // React StrictMode cancels its first mount's request; only live requests can supply the preview.
  return read.mock.calls.filter(([, , signal]) => !signal?.aborted);
}

describe('artifact content previews', () => {
  it('loads only after opening, preserves sample order, and shows dimensions and dtypes', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(dataset));
    const rawRead = vi.spyOn(Apis, 'readFile');
    const { rerender } = render(view([artifact], false));
    expect(read).not.toHaveBeenCalled();
    rerender(view());
    const table = await screen.findByRole('table', { name: 'Dataset sample' });
    expect(within(table).getAllByRole('row')).toHaveLength(11);
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('0.1');
    expect(within(table).getAllByRole('row')[2]).toHaveTextContent('0.8');
    expect(screen.getByRole('status')).toHaveTextContent('Showing 10 of 1000 rows; 3 of 3 columns');
    expect(screen.getByText('float64')).toBeVisible();
    expect(screen.getByText('Only sampled rows are shown.')).toBeVisible();
    expect(completedRequests(read)).toEqual([
      ['dataset-1', 'team-a', expect.objectContaining({ aborted: false })],
    ]);
    expect(rawRead).not.toHaveBeenCalled();
  });

  it('keeps missing and malformed declarations out of the network path', () => {
    const read = vi.spyOn(Apis, 'getArtifactContentPreview');
    const { rerender } = render(view([{ ...artifact, metadata: {} }]));
    expect(screen.getByText(/pipeline and component preview settings/)).toBeVisible();
    rerender(
      view([
        { ...artifact, metadata: { _kfp_preview: { version: 2, path: '_kfp_preview.json' } } },
      ]),
    );
    expect(screen.getByText('Invalid artifact preview declaration.')).toBeVisible();
    expect(read).not.toHaveBeenCalled();
  });

  it('preserves multi-artifact selection across refreshes and reordering', async () => {
    const second: V2Artifact = { ...artifact, artifact_id: 'dataset-2', name: 'second data' };
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(dataset));
    const { rerender } = render(view([artifact, second]));
    expect(read).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Artifact preview' }));
    fireEvent.click(screen.getByRole('option', { name: 'second data' }));
    await screen.findByRole('table', { name: 'Dataset sample' });
    expect(completedRequests(read)).toEqual([
      ['dataset-2', 'team-a', expect.objectContaining({ aborted: false })],
    ]);
    const attempts = read.mock.calls.length;
    rerender(view([{ ...second }, { ...artifact }]));
    expect(screen.getByRole('combobox')).toHaveTextContent('second data');
    expect(read).toHaveBeenCalledTimes(attempts);
  });

  it('does not refetch completion updates until the visualization is visible', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(dataset));
    const { rerender } = render(view());
    await screen.findByRole('table', { name: 'Dataset sample' });
    rerender(view([artifact], false, true));
    expect(completedRequests(read)).toHaveLength(1);
    rerender(view([artifact], true, true));
    await waitFor(() => expect(completedRequests(read)).toHaveLength(2));
  });

  it('cancels in-flight reads when the preview is closed', async () => {
    let requestSignal: AbortSignal | undefined;
    vi.spyOn(Apis, 'getArtifactContentPreview').mockImplementation((_id, _namespace, signal) => {
      requestSignal = signal;
      return new Promise((_resolve, reject) => {
        signal?.addEventListener('abort', () => reject(new Error('cancelled')), { once: true });
      });
    });
    const { rerender } = render(view());
    await screen.findByText('Artifact preview is loading.');
    rerender(view([artifact], false));
    expect(requestSignal?.aborted).toBe(true);
  });

  it('separates cached previews by artifact and namespace', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(dataset));
    const { rerender } = render(view());
    await screen.findByRole('table', { name: 'Dataset sample' });
    rerender(view([{ ...artifact, namespace: 'team-b' }]));
    await waitFor(() => expect(completedRequests(read)).toHaveLength(2));
    expect(read).toHaveBeenLastCalledWith(
      'dataset-1',
      'team-b',
      expect.objectContaining({ aborted: false }),
    );
  });

  it('shows read errors with an explicit retry and rejects malformed files', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockRejectedValue(new Error('_kfp_preview.json is missing'));
    render(view());
    await screen.findByText('Unable to display the artifact preview.');
    expect(completedRequests(read)).toHaveLength(1);
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
      JSON.stringify({
        ...dataset,
        rows: [['<script>alert(1)</script>', null, '']],
      }),
    );
    const { container } = render(view());
    expect(await screen.findByText('<script>alert(1)</script>')).toBeVisible();
    expect(screen.getByText('null')).toBeVisible();
    expect(screen.getByText('""')).toBeVisible();
    expect(container.querySelector('script')).toBeNull();
  });

  it('renders model identity, recorded parameters, and structural facts separately', async () => {
    vi.spyOn(Apis, 'getArtifactContentPreview').mockResolvedValue(JSON.stringify(model));
    render(view([{ ...artifact, type: ArtifactArtifactType.Model }]));
    expect(await screen.findByRole('region', { name: 'Model preview' })).toBeVisible();
    expect(screen.getByText('xgboost.XGBClassifier')).toBeVisible();
    expect(screen.getByRole('heading', { name: 'Recorded hyperparameters' })).toBeVisible();
    expect(screen.getByRole('heading', { name: 'Structural information' })).toBeVisible();
    expect(screen.getByText('max_depth')).toBeVisible();
    expect(screen.getByText('tree_count')).toBeVisible();
    const parameters = screen.getByRole('table', { name: 'Recorded hyperparameters' });
    const structure = screen.getByRole('table', { name: 'Structural information' });
    expect(within(parameters).getAllByRole('row')).toHaveLength(
      Object.keys(model.hyperparameters).length + 1,
    );
    expect(
      within(parameters)
        .getAllByRole('rowheader')
        .map((cell) => cell.textContent),
    ).toEqual(Object.keys(model.hyperparameters).sort());
    expect(within(structure).getAllByRole('row')).toHaveLength(
      Object.keys(model.structure).length + 1,
    );
    expect(screen.queryByText('Rows per page:')).toBeNull();
  });

  it('keeps every recorded model entry visible without spacer rows or pagination', async () => {
    const hyperparameters = Object.fromEntries(
      Array.from({ length: 100 }, (_, index) => [`parameter_${index}`, index]),
    );
    vi.spyOn(Apis, 'getArtifactContentPreview').mockResolvedValue(
      JSON.stringify({ ...model, hyperparameters, structure: {} }),
    );
    render(view([{ ...artifact, type: ArtifactArtifactType.Model }]));
    const table = await screen.findByRole('table', { name: 'Recorded hyperparameters' });
    expect(within(table).getAllByRole('row')).toHaveLength(101);
    expect(within(table).getAllByRole('rowheader')).toHaveLength(100);
    expect(screen.getByText('Not recorded.')).toBeVisible();
  });
});
