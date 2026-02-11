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

import * as React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import { NewExperiment } from './NewExperiment';
import TestUtils from 'src/TestUtils';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';
import { logger } from 'src/lib/Utils';

describe('NewExperiment', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let newExperimentRef: React.RefObject<NewExperiment> | null = null;

  const createExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'createExperiment');
  const listPipelineVersionsSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
  let historyPushSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let updateBannerSpy: ReturnType<typeof vi.fn>;

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: RoutePage.NEW_EXPERIMENT, search: '' } as any,
      match: '' as any,
      toolbarProps: NewExperiment.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    } as PageProps;
  }

  function getInstance(): NewExperiment {
    if (!newExperimentRef?.current) {
      throw new Error('NewExperiment instance not available');
    }
    return newExperimentRef.current;
  }

  async function renderNewExperiment(
    propsPatch: Partial<PageProps & { namespace?: string }> = {},
  ): Promise<void> {
    newExperimentRef = React.createRef<NewExperiment>();
    const props = { ...generateProps(), ...propsPatch } as PageProps;
    renderResult = render(<NewExperiment ref={newExperimentRef} {...props} />);
    await TestUtils.flushPromises();
  }

  function fillExperimentName(value: string) {
    fireEvent.change(screen.getByLabelText(/Experiment name/i), {
      target: { value },
    });
  }

  function fillDescription(value: string) {
    fireEvent.change(screen.getByLabelText(/Description/i), {
      target: { value },
    });
  }

  function getNextButton(): HTMLButtonElement {
    return screen.getByRole('button', { name: 'Next' });
  }

  function getCancelButton(): HTMLButtonElement {
    return screen.getByRole('button', { name: 'Cancel' });
  }

  beforeEach(() => {
    historyPushSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    updateBannerSpy = vi.fn();
    createExperimentSpy.mockResolvedValue({ experiment_id: 'new-experiment-id' } as any);
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] } as any);
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    newExperimentRef = null;
    vi.clearAllMocks();
  });

  it('renders the new experiment page', async () => {
    await renderNewExperiment();
    await waitFor(() =>
      expect(screen.getByText('Experiment name is required')).toBeInTheDocument(),
    );
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('does not include any action buttons in the toolbar', async () => {
    await renderNewExperiment();
    expect(updateToolbarSpy).toHaveBeenCalledWith({
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'New experiment',
    });
  });

  it("enables the 'Next' button when an experiment name is entered", async () => {
    await renderNewExperiment();
    expect(getNextButton()).toBeDisabled();

    fillExperimentName('experiment name');

    await waitFor(() => expect(getNextButton()).not.toBeDisabled());
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it("re-disables the 'Next' button when an experiment name is cleared after having been entered", async () => {
    await renderNewExperiment();
    expect(getNextButton()).toBeDisabled();

    fillExperimentName('experiment name');
    await waitFor(() => expect(getNextButton()).not.toBeDisabled());

    fillExperimentName('');
    await waitFor(() => expect(getNextButton()).toBeDisabled());
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('updates the experiment name', async () => {
    await renderNewExperiment();
    fillExperimentName('experiment name');

    await waitFor(() => {
      expect(getInstance().state).toEqual({
        description: '',
        experimentName: 'experiment name',
        isbeingCreated: false,
        validationError: '',
      });
    });
  });

  it('updates the experiment description', async () => {
    await renderNewExperiment();
    fillDescription('a description!');

    await waitFor(() => {
      expect(getInstance().state).toEqual({
        description: 'a description!',
        experimentName: '',
        isbeingCreated: false,
        validationError: 'Experiment name is required',
      });
    });
  });

  it("sets the page to a busy state upon clicking 'Next'", async () => {
    await renderNewExperiment();

    fillExperimentName('experiment-name');
    await waitFor(() => expect(getNextButton()).not.toBeDisabled());

    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    expect(getInstance().state).toHaveProperty('isbeingCreated', true);
    expect(getNextButton()).toBeDisabled();
  });

  it("calls the createExperiment API with the new experiment upon clicking 'Next'", async () => {
    await renderNewExperiment();

    fillExperimentName('experiment name');
    fillDescription('experiment description');

    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    expect(createExperimentSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        description: 'experiment description',
        display_name: 'experiment name',
      }),
    );
  });

  it('calls the createExperimentAPI with namespace when it is provided', async () => {
    await renderNewExperiment({ namespace: 'test-ns' });

    fillExperimentName('a-random-experiment-name-DO-NOT-VERIFY-THIS');
    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    expect(createExperimentSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        namespace: 'test-ns',
      }),
    );
  });

  it('navigates to NewRun page upon successful creation', async () => {
    const experimentId = 'test-exp-id-1';
    createExperimentSpy.mockResolvedValue({ experiment_id: experimentId } as any);
    await renderNewExperiment();

    fillExperimentName('experiment-name');
    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?experimentId=${experimentId}` + `&firstRunInExperiment=1`,
    );
  });

  it('includes pipeline ID and version ID in NewRun page query params if present', async () => {
    const experimentId = 'test-exp-id-1';
    createExperimentSpy.mockResolvedValue({ experiment_id: experimentId } as any);

    const pipelineId = 'some-pipeline-id';
    const pipelineVersionId = 'version-id';
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [{ pipeline_version_id: pipelineVersionId }],
    } as any);

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipelineId}`;
    await renderNewExperiment(props as any);

    await waitFor(() => expect(getInstance().state.pipelineId).toBe(pipelineId));

    fillExperimentName('experiment-name');
    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN +
        `?experimentId=${experimentId}` +
        `&pipelineId=${pipelineId}` +
        `&pipelineVersionId=${pipelineVersionId}` +
        `&firstRunInExperiment=1`,
    );
  });

  it('shows snackbar confirmation after experiment is created', async () => {
    await renderNewExperiment();

    fillExperimentName('experiment-name');
    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      autoHideDuration: 10000,
      message: 'Successfully created new Experiment: experiment-name',
      open: true,
    });
  });

  it('unsets busy state when creation fails', async () => {
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => null);
    await renderNewExperiment();

    fillExperimentName('experiment-name');

    TestUtils.makeErrorResponseOnce(createExperimentSpy as any, 'test error!');
    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    await waitFor(() => expect(getInstance().state).toHaveProperty('isbeingCreated', false));
    expect(loggerErrorSpy).toHaveBeenCalled();
  });

  it('shows error dialog when creation fails', async () => {
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => null);
    await renderNewExperiment();

    fillExperimentName('experiment-name');

    TestUtils.makeErrorResponseOnce(createExperimentSpy as any, 'test error!');
    fireEvent.click(getNextButton());
    await TestUtils.flushPromises();

    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'Experiment creation failed');
    expect(call).toHaveProperty('content', 'test error!');
    expect(loggerErrorSpy).toHaveBeenCalled();
  });

  it('navigates to experiment list page upon cancellation', async () => {
    await renderNewExperiment();
    fireEvent.click(getCancelButton());
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.EXPERIMENTS);
  });
});
