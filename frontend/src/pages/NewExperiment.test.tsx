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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import NewExperiment from './NewExperiment';
import { CommonTestWrapper } from '../TestWrapper';
import TestUtils from '../TestUtils';

// Mock React Router hooks
const mockNavigate = jest.fn();
const mockLocation = { pathname: '', search: '', hash: '', state: null, key: 'default' };
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
  useLocation: () => mockLocation,
}));

describe('NewExperiment', () => {
  const TEST_EXPERIMENT_ID = 'new-experiment-id';
  const createExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'createExperiment');
  const navigateSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  function generateProps(): PageProps {
    return {
      navigate: navigateSpy,
      location: {
        pathname: RoutePage.NEW_EXPERIMENT,
        search: '',
        hash: '',
        state: null,
        key: 'default',
      },
      match: { params: {}, isExact: true, path: '', url: '' },
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'New experiment' },
      updateBanner: () => null,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
    mockNavigate.mockClear();
    navigateSpy.mockClear();

    createExperimentSpy.mockResolvedValue({
      experiment_id: TEST_EXPERIMENT_ID,
      display_name: 'new-experiment-name',
    });
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it('renders the new experiment page', () => {
    const { container } = render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );
    expect(container.firstChild).toMatchSnapshot();
  });

  it('does not include any action buttons in the toolbar', () => {
    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
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
        <NewExperiment {...generateProps()} />
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
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    expect(nextButton.closest('button')?.disabled).toEqual(false);

    // Clear experiment name
    fireEvent.change(experimentNameInput, { target: { value: '' } });
    expect(nextButton.closest('button')?.disabled).toEqual(true);
  });

  it('updates the experiment name', () => {
    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    expect(experimentNameInput.closest('input')?.value).toBe('new-experiment-name');
  });

  it('updates the experiment description', () => {
    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentDescriptionInput = screen.getByLabelText('Description');
    fireEvent.change(experimentDescriptionInput, {
      target: { value: 'new-experiment-description' },
    });
    expect(experimentDescriptionInput.closest('textarea')?.value).toBe(
      'new-experiment-description',
    );
  });

  it("sets the page to a busy state upon clicking 'Next'", async () => {
    // Create a longer-running promise to test busy state
    let resolvePromise: (value: any) => void;
    const pendingPromise = new Promise<any>(resolve => {
      resolvePromise = resolve;
    });
    createExperimentSpy.mockReturnValue(pendingPromise);

    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    // Check that button is in busy state
    expect(nextButton.closest('button')?.disabled).toEqual(true);

    // Resolve the promise to clean up
    resolvePromise!({
      experiment_id: TEST_EXPERIMENT_ID,
      display_name: 'new-experiment-name',
    });
    await TestUtils.flushPromises();
  });

  it("calls the createExperiment API with the new experiment upon clicking 'Next'", async () => {
    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const experimentDescriptionInput = screen.getByLabelText('Description');
    fireEvent.change(experimentDescriptionInput, {
      target: { value: 'new-experiment-description' },
    });
    const nextButton = screen.getByText('Next');
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

  it('calls the createExperimentAPI with namespace when it is provided', async () => {
    // Mock the NamespaceContext to provide a test namespace
    const mockUseContext = jest.spyOn(React, 'useContext');
    mockUseContext.mockImplementation(context => {
      // Return test namespace for NamespaceContext, otherwise use real implementation
      if (
        context.displayName === 'NamespaceContext' ||
        context === require('src/lib/KubeflowClient').NamespaceContext
      ) {
        return 'test-namespace';
      }
      return jest.requireActual('react').useContext(context);
    });

    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '',
          display_name: 'new-experiment-name',
          namespace: 'test-namespace',
        }),
      );
    });

    mockUseContext.mockRestore();
  });

  it('navigates to NewRun page upon successful creation', async () => {
    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(navigateSpy).toHaveBeenCalledWith(
        RoutePage.NEW_RUN + `?experimentId=${TEST_EXPERIMENT_ID}&firstRunInExperiment=1`,
      );
    });
  });

  it('includes pipeline ID and version ID in NewRun page query params if present', async () => {
    const TEST_PIPELINE_ID = 'test-pipeline-id';
    const TEST_PIPELINE_VERSION_ID = 'test-pipeline-version-id';

    // Mock the getLatestVersion call
    const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [{ pipeline_version_id: TEST_PIPELINE_VERSION_ID }],
      total_size: 1,
    });

    // Update the location mock to include the pipeline ID in search params
    mockLocation.search = `?${QUERY_PARAMS.pipelineId}=${TEST_PIPELINE_ID}`;

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${TEST_PIPELINE_ID}`;

    render(
      <CommonTestWrapper>
        <NewExperiment {...props} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(navigateSpy).toHaveBeenCalledWith(
        RoutePage.NEW_RUN +
          `?experimentId=${TEST_EXPERIMENT_ID}&pipelineId=${TEST_PIPELINE_ID}&pipelineVersionId=${TEST_PIPELINE_VERSION_ID}&firstRunInExperiment=1`,
      );
    });

    // Clean up the mock
    mockLocation.search = '';
  });

  it('shows snackbar confirmation after experiment is created', async () => {
    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(updateSnackbarSpy).toHaveBeenCalledWith({
        autoHideDuration: 10000,
        message: 'Successfully created new Experiment: new-experiment-name',
        open: true,
      });
    });
  });

  it('unsets busy state when creation fails', async () => {
    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'Creation failed');

    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(createExperimentSpy).toHaveBeenCalled();
    });

    // After error, the dialog should appear and closing it should unset busy state
    expect(updateDialogSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        buttons: [{ text: 'Dismiss' }],
        title: 'Experiment creation failed',
      }),
    );

    // Simulate dismissing the dialog
    const dialogCall = updateDialogSpy.mock.calls[0][0];
    if (dialogCall.onClose) {
      dialogCall.onClose();
    }

    // Button should be enabled again after dismissing error dialog
    await waitFor(() => {
      expect(nextButton.closest('button')?.disabled).toEqual(false);
    });
  });

  it('shows error dialog when creation fails', async () => {
    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'There was something wrong!');

    render(
      <CommonTestWrapper>
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const experimentNameInput = screen.getByLabelText(/Experiment name/);
    fireEvent.change(experimentNameInput, { target: { value: 'new-experiment-name' } });
    const nextButton = screen.getByText('Next');
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
        <NewExperiment {...generateProps()} />
      </CommonTestWrapper>,
    );

    const cancelButton = screen.getByText('Cancel');
    fireEvent.click(cancelButton);

    expect(navigateSpy).toHaveBeenCalledWith(RoutePage.EXPERIMENTS);
  });
});
