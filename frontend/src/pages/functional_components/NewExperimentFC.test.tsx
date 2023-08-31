/*
 * Copyright 2023 The Kubeflow Authors
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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import TestUtils from 'src/TestUtils';
import { NewExperimentFC } from './NewExperimentFC';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import * as features from 'src/features';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';

describe('NewExperiment', () => {
  const TEST_EXPERIMENT_ID = 'new-experiment-id';
  const createExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'createExperiment');
  const historyPushSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: RoutePage.NEW_EXPERIMENT } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: TEST_EXPERIMENT_ID },
      updateBanner: () => null,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
    // mock both v2_alpha and functional feature keys are enable.
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(true);

    createExperimentSpy.mockImplementation(() => ({
      experiment_id: 'new-experiment-id',
      display_name: 'new-experiment-name',
    }));
  });

  it('does not include any action buttons in the toolbar', () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    expect(updateToolbarSpy).toHaveBeenCalledWith({
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'New experiment',
    });
  });

  it("enables the 'Next' button when an experiment name is entered", () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);
  });

  it("re-disables the 'Next' button when an experiment name is cleared after having been entered", () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    // Remove experiment name
    fireEvent.change(experimentNameInput, { target: { value: '' } });
    expect(nextButton.closest('button')?.disabled).toEqual(true);
  });

  it('updates the experiment name', () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    expect(experimentNameInput.closest('input')?.value).toBe('new-experiment-name');
  });

  it('create new experiment', async () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const experimentDescriptionInput = screen.getByLabelText('Description');
    fireEvent.change(experimentDescriptionInput, {
      target: { value: 'new-experiment-description' },
    });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    fireEvent.click(nextButton);
    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          description: 'new-experiment-description',
          display_name: 'new-experiment-name',
        }),
      );
    });
  });

  it('create new experiment with namespace provided', async () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} namespace='test-ns' />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    fireEvent.click(nextButton);
    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '',
          display_name: 'new-experiment-name',
          namespace: 'test-ns',
        }),
      );
    });
  });

  it('navigates to NewRun page upon successful creation', async () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    fireEvent.click(nextButton);
    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '',
          display_name: 'new-experiment-name',
        }),
      );
    });
    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?experimentId=${TEST_EXPERIMENT_ID}` + `&firstRunInExperiment=1`,
    );
  });

  it('includes pipeline ID and version ID in NewRun page query params if present', async () => {
    const pipelineId = 'some-pipeline-id';
    const pipelineVersionId = 'version-id';
    const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
    listPipelineVersionsSpy.mockImplementation(() => ({
      pipeline_versions: [{ pipeline_version_id: pipelineVersionId }],
    }));

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipelineId}`;

    render(
      <CommonTestWrapper>
        <NewExperimentFC {...props} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    fireEvent.click(nextButton);
    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '',
          display_name: 'new-experiment-name',
        }),
      );
    });

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN +
        `?experimentId=${TEST_EXPERIMENT_ID}` +
        `&pipelineId=${pipelineId}` +
        `&pipelineVersionId=${pipelineVersionId}` +
        `&firstRunInExperiment=1`,
    );
  });

  it('shows snackbar confirmation after experiment is created', async () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    fireEvent.click(nextButton);
    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '',
          display_name: 'new-experiment-name',
        }),
      );
    });
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      autoHideDuration: 10000,
      message: 'Successfully created new Experiment: new-experiment-name',
      open: true,
    });
  });

  it('shows error dialog when experiment creation fails', async () => {
    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'There was something wrong!');
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    fireEvent.click(nextButton);
    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalled();
    });

    expect(updateDialogSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        content: 'There was something wrong!',
        title: 'Experiment creation failed',
      }),
    );
  });

  it('navigates to experiment list page upon cancellation', () => {
    render(
      <CommonTestWrapper>
        <NewExperimentFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    const cancelButton = screen.getByText('Cancel');
    fireEvent.click(cancelButton);

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.EXPERIMENTS);
  });
});
