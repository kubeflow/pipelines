/*
 * Copyright 2018 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { createRef, RefObject } from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { load } from 'js-yaml';
import { Apis } from 'src/lib/Apis';
import TestUtils, { mockResizeObserver } from 'src/TestUtils';
import { RouteParams } from 'src/components/Router';
import PipelineDetails from './PipelineDetails';
import template from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';

beforeEach(() => {
  mockResizeObserver();
  vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline').mockResolvedValue({ pipeline_id: 'pipeline' });
  vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions').mockResolvedValue({
    pipeline_versions: [],
  });
});
afterEach(() => vi.restoreAllMocks());
function renderSpec(pipeline_spec: object | undefined, ref?: RefObject<PipelineDetails | null>) {
  vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion').mockResolvedValue({
    pipeline_id: 'pipeline',
    pipeline_version_id: 'version',
    pipeline_spec,
  });
  const props = TestUtils.generatePageProps(
    PipelineDetails,
    { search: '' } as any,
    { [RouteParams.pipelineId]: 'pipeline', [RouteParams.pipelineVersionId]: 'version' },
    vi.fn(),
    vi.fn(),
    vi.fn(),
    vi.fn(),
    vi.fn(),
  );
  render(
    <MemoryRouter>
      <PipelineDetails {...props} ref={ref} />
    </MemoryRouter>,
  );
  return props;
}
it('renders native IR with feature flags disabled', async () => {
  window.__FEATURE_FLAGS__ = '[]';
  const props = renderSpec(load(template) as object);
  await screen.findByTestId('pipeline-detail-v2');
  expect(props.updateBanner).not.toHaveBeenCalledWith(expect.objectContaining({ mode: 'error' }));
});
it.each([{}, { kind: 'Workflow', apiVersion: 'argoproj.io/v1alpha1', spec: {} }])(
  'rejects non-IR specs without rendering a legacy graph',
  async (spec) => {
    const props = renderSpec(spec);
    await waitFor(() =>
      expect(props.updateBanner).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining('failed to generate Pipeline graph'),
          mode: 'error',
        }),
      ),
    );
    expect(screen.queryByTestId('pipeline-detail-v1')).toBeNull();
  },
);

it('warns for a missing spec on load and selection, and clears it for valid IR', async () => {
  const spec = load(template) as object;
  vi.mocked(Apis.pipelineServiceApiV2.listPipelineVersions).mockResolvedValue({
    pipeline_versions: [
      { pipeline_id: 'pipeline', pipeline_version_id: 'version' },
      { pipeline_id: 'pipeline', pipeline_version_id: 'valid', pipeline_spec: spec },
    ],
  });
  const ref = createRef<PipelineDetails>();
  const props = renderSpec(undefined, ref);
  const warning = expect.objectContaining({
    mode: 'warning',
    message: expect.stringContaining('no pipeline spec'),
  });
  await waitFor(() => expect(props.updateBanner).toHaveBeenCalledWith(warning));
  await act(async () => {
    await ref.current!.handleVersionSelected('valid');
  });
  expect(props.updateBanner).toHaveBeenLastCalledWith({});
  await act(async () => {
    await ref.current!.handleVersionSelected('version');
  });
  expect(props.updateBanner).toHaveBeenLastCalledWith(warning);
});

it('preserves the selected tab on refresh and resets graph state only on version change', async () => {
  const spec = load(template) as object;
  vi.mocked(Apis.pipelineServiceApiV2.listPipelineVersions).mockResolvedValue({
    pipeline_versions: [
      { pipeline_id: 'pipeline', pipeline_version_id: 'version', pipeline_spec: spec },
      { pipeline_id: 'pipeline', pipeline_version_id: 'version-2', pipeline_spec: spec },
    ],
  });
  const ref = createRef<PipelineDetails>();
  renderSpec(spec, ref);
  await screen.findByTestId('DagCanvas');
  fireEvent.click(screen.getByRole('button', { name: 'Pipeline Spec' }));
  expect(screen.queryByTestId('DagCanvas')).toBeNull();
  await act(async () => {
    await ref.current!.refresh();
  });
  expect(screen.queryByTestId('DagCanvas')).toBeNull();
  await act(async () => {
    await ref.current!.handleVersionSelected('version-2');
  });
  expect(screen.getByTestId('DagCanvas')).toBeInTheDocument();
});
