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
import { NewPipelineVersionFC } from 'src/pages/functional_components/NewPipelineVersionFC';
import TestUtils from 'src/TestUtils';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';

describe('NewPipelineVersion', () => {
  const PIPELINE_ID = 'original-run-pipeline-id';
  const PIPELINE_VERSION_ID = 'original-run-pipeline-version-id';
  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
  const uploadPipelineSpy = jest.spyOn(Apis, 'uploadPipelineV2');
  const createPipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'createPipeline');
  const uploadPipelineVersionSpy = jest.spyOn(Apis, 'uploadPipelineVersionV2');
  const createPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'createPipelineVersion');
  const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');

  let MOCK_PIPELINE = {
    pipeline_id: PIPELINE_ID,
    display_name: 'original mock pipeline name',
  };

  let MOCK_PIPELINE_VERSION = {
    pipeline_id: PIPELINE_ID,
    pipeline_version_id: PIPELINE_VERSION_ID,
    display_name: 'original mock pipeline version name',
    description: 'original mock pipeline version description',
  };

  function generateProps(search?: string): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: { pathname: RoutePage.NEW_PIPELINE_VERSION, search: search } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'New Pipeline' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
    getPipelineSpy.mockImplementation(() => MOCK_PIPELINE);
    uploadPipelineSpy.mockImplementation(() => MOCK_PIPELINE);
    createPipelineSpy.mockImplementation(() => MOCK_PIPELINE);
    uploadPipelineVersionSpy.mockImplementation(() => MOCK_PIPELINE_VERSION);
    createPipelineVersionSpy.mockImplementation(() => MOCK_PIPELINE_VERSION);
    listPipelineVersionsSpy.mockImplementation(() => ({
      pipeline_versions: [MOCK_PIPELINE_VERSION],
    }));
  });

  it('does not include any action buttons in the toolbar', () => {
    render(
      <CommonTestWrapper>
        <NewPipelineVersionFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    expect(updateToolbarSpy).toHaveBeenCalledWith({
      actions: {},
      breadcrumbs: [{ displayName: 'Pipeline Versions', href: '/pipeline_versions/new' }],
      pageTitle: 'New Pipeline',
    });
  });

  describe('switching between creating pipeline and creating pipeline version', () => {
    it('creates pipeline is default when landing from pipeline list page', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalledTimes(0);
      });

      screen.getByText('Upload pipeline with the specified package.');
    });

    it('creates pipeline version is default when landing from pipeline details page', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
          />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
      });

      screen.getByText('Upload pipeline version with the specified package.');
    });
  });

  describe('create new pipeline', () => {
    it('switches between import methods', () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} />
        </CommonTestWrapper>,
      );

      screen.getByText('URL must be publicly accessible.');

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      screen.getByText(/Choose a pipeline package file from your computer/);
    });

    it('creates pipeline from url in single user mode', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} buildInfo={{ apiServerMultiUser: false }} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const pipelineDescriptionInput = screen.getByLabelText('Pipeline Description');
      fireEvent.change(pipelineDescriptionInput, {
        target: { value: 'test-pipeline-description' },
      });

      const urlInput = screen.getByLabelText('Package Url');
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(createPipelineSpy).toHaveBeenCalledWith({
          description: 'test-pipeline-description',
          display_name: 'test-pipeline-name',
        });
        expect(createPipelineVersionSpy).toHaveBeenCalledWith('original-run-pipeline-id', {
          pipeline_id: 'original-run-pipeline-id',
          display_name: '',
          description: '',
          package_url: {
            pipeline_url: 'https://dummy_package_url',
          },
        });
      });

      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
    });

    it('creates private pipeline from url in multi user mode', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps()}
            buildInfo={{ apiServerMultiUser: true }}
            namespace='ns'
          />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const pipelineDescriptionInput = screen.getByLabelText('Pipeline Description');
      fireEvent.change(pipelineDescriptionInput, {
        target: { value: 'test-pipeline-description' },
      });

      const urlInput = screen.getByLabelText('Package Url');
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(createPipelineSpy).toHaveBeenCalledWith({
          description: 'test-pipeline-description',
          display_name: 'test-pipeline-name',
          namespace: 'ns',
        });

        expect(createPipelineVersionSpy).toHaveBeenCalledWith('original-run-pipeline-id', {
          pipeline_id: 'original-run-pipeline-id',
          display_name: '',
          description: '',
          package_url: {
            pipeline_url: 'https://dummy_package_url',
          },
        });
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
    });

    it('creates shared pipeline from url in multi user mode', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} buildInfo={{ apiServerMultiUser: true }} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const pipelineDescriptionInput = screen.getByLabelText('Pipeline Description');
      fireEvent.change(pipelineDescriptionInput, {
        target: { value: 'test-pipeline-description' },
      });

      const urlInput = screen.getByLabelText('Package Url');
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(createPipelineSpy).toHaveBeenCalledWith({
          description: 'test-pipeline-description',
          display_name: 'test-pipeline-name',
        });

        expect(createPipelineVersionSpy).toHaveBeenCalledWith('original-run-pipeline-id', {
          pipeline_id: 'original-run-pipeline-id',
          display_name: '',
          description: '',
          package_url: {
            pipeline_url: 'https://dummy_package_url',
          },
        });
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
    });

    it('shows error dialog when create pipeline from URL fails', async () => {
      TestUtils.makeErrorResponseOnce(createPipelineSpy, 'There was something wrong!');
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const urlInput = screen.getByLabelText('Package Url');
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(createPipelineSpy).toHaveBeenCalled();
        expect(createPipelineVersionSpy).toHaveBeenCalledTimes(0);
      });

      expect(updateDialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'There was something wrong!',
          title: 'Pipeline creation failed',
        }),
      );
    });

    it('creates pipeline from local file in single user mode', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} buildInfo={{ apiServerMultiUser: false }} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const pipelineDescriptionInput = screen.getByLabelText('Pipeline Description');
      fireEvent.change(pipelineDescriptionInput, {
        target: { value: 'test-pipeline-description' },
      });

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      // mock drop file from local.
      const uploadFile = await screen.findByText(/File/);
      const file = new File(['file contents'], 'test-pipeline.yaml', { type: 'text/yaml' });
      Object.defineProperty(uploadFile, 'files', {
        value: [file],
      });
      fireEvent.drop(uploadFile);

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(uploadPipelineSpy).toHaveBeenCalledWith(
          'test-pipeline-name',
          'test-pipeline-description',
          file,
          undefined,
        );
        expect(listPipelineVersionsSpy).toHaveBeenCalled();
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
    });

    it('creates private pipeline from local file in multi user mode', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps()}
            buildInfo={{ apiServerMultiUser: true }}
            namespace='ns'
          />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const pipelineDescriptionInput = screen.getByLabelText('Pipeline Description');
      fireEvent.change(pipelineDescriptionInput, {
        target: { value: 'test-pipeline-description' },
      });

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      // mock drop file from local.
      const uploadFile = await screen.findByText(/File/);
      const file = new File(['file contents'], 'test-pipeline.yaml', { type: 'text/yaml' });
      Object.defineProperty(uploadFile, 'files', {
        value: [file],
      });
      fireEvent.drop(uploadFile);

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(uploadPipelineSpy).toHaveBeenCalledWith(
          'test-pipeline-name',
          'test-pipeline-description',
          file,
          'ns',
        );
        expect(listPipelineVersionsSpy).toHaveBeenCalled();
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
    });

    it('creates shared pipeline from local file in multi user mode', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} buildInfo={{ apiServerMultiUser: true }} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const pipelineDescriptionInput = screen.getByLabelText('Pipeline Description');
      fireEvent.change(pipelineDescriptionInput, {
        target: { value: 'test-pipeline-description' },
      });

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      // mock drop file from local.
      const uploadFile = await screen.findByText(/File/);
      const file = new File(['file contents'], 'test-pipeline.yaml', { type: 'text/yaml' });
      Object.defineProperty(uploadFile, 'files', {
        value: [file],
      });
      fireEvent.drop(uploadFile);

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(uploadPipelineSpy).toHaveBeenCalledWith(
          'test-pipeline-name',
          'test-pipeline-description',
          file,
          undefined,
        );
        expect(listPipelineVersionsSpy).toHaveBeenCalled();
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
    });

    it('shows error dialog when create pipeline from local file fails', async () => {
      TestUtils.makeErrorResponseOnce(uploadPipelineSpy, 'There was something wrong!');
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      // mock drop file from local.
      const uploadFile = await screen.findByText(/File/);
      const file = new File(['file contents'], 'test-pipeline.yaml', { type: 'text/yaml' });
      Object.defineProperty(uploadFile, 'files', {
        value: [file],
      });
      fireEvent.drop(uploadFile);

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(uploadPipelineSpy).toHaveBeenCalled();
        expect(listPipelineVersionsSpy).toHaveBeenCalledTimes(0);
      });

      expect(updateDialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'There was something wrong!',
          title: 'Pipeline creation failed',
        }),
      );
    });
  });

  describe('creating version under an existing pipeline', () => {
    it('allows updating pipeline version name', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
          />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
      });

      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-version-name' } });
      expect(pipelineVersionNameInput.closest('input')?.value).toBe('new-version-name');
    });

    it('creates pipeline version from url under an existing pipeline', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
          />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
      });

      // Change pipeline version name
      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-version-name' } });

      const urlInput = screen.getByLabelText('Package Url');
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(createPipelineVersionSpy).toHaveBeenCalledWith('original-run-pipeline-id', {
          pipeline_id: 'original-run-pipeline-id',
          display_name: 'new-version-name',
          description: '',
          package_url: {
            pipeline_url: 'https://dummy_package_url',
          },
        });
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
      expect(updateSnackbarSpy).toHaveBeenCalledWith({
        autoHideDuration: 10000,
        message: `Successfully created new pipeline version: ${MOCK_PIPELINE_VERSION.display_name}`,
        open: true,
      });
    });

    it('shows error dialog when create pipeline version from URL fails', async () => {
      TestUtils.makeErrorResponseOnce(createPipelineVersionSpy, 'There was something wrong!');
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC {...generateProps()} />
        </CommonTestWrapper>,
      );

      const pipelineNameInput = screen.getByLabelText(/Pipeline Name/);
      fireEvent.change(pipelineNameInput, { target: { value: 'test-pipeline-name' } });

      const urlInput = screen.getByLabelText('Package Url');
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(createPipelineVersionSpy).toHaveBeenCalled();
      });

      expect(updateDialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'There was something wrong!',
          title: 'Pipeline version creation failed',
        }),
      );
    });

    it('creates pipeline version from local file under an existing pipeline', async () => {
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
          />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
      });

      // Change pipeline version name
      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-version-name' } });

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      // mock drop file from local.
      const uploadFile = await screen.findByText(/File/);
      const file = new File(['file contents'], 'test-pipeline.yaml', { type: 'text/yaml' });
      Object.defineProperty(uploadFile, 'files', {
        value: [file],
      });
      fireEvent.drop(uploadFile);

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(uploadPipelineVersionSpy).toHaveBeenCalledWith(
          'new-version-name',
          'original-run-pipeline-id',
          file,
          '',
        );
      });
      expect(historyPushSpy).toHaveBeenCalledWith(
        `/pipelines/details/${PIPELINE_ID}/version/${PIPELINE_VERSION_ID}?`,
      );
      expect(updateSnackbarSpy).toHaveBeenCalledWith({
        autoHideDuration: 10000,
        message: `Successfully created new pipeline version: ${MOCK_PIPELINE_VERSION.display_name}`,
        open: true,
      });
    });

    it('shows error dialog when create pipeline version from local file fails', async () => {
      TestUtils.makeErrorResponseOnce(uploadPipelineVersionSpy, 'There was something wrong!');
      render(
        <CommonTestWrapper>
          <NewPipelineVersionFC
            {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
          />
        </CommonTestWrapper>,
      );

      await waitFor(() => {
        expect(getPipelineSpy).toHaveBeenCalled();
      });

      // Change pipeline version name
      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-version-name' } });

      // switch to upload file method
      const uploadFileBtn = screen.getByText('Upload a file');
      fireEvent.click(uploadFileBtn);

      // mock drop file from local.
      const uploadFile = await screen.findByText(/File/);
      const file = new File(['file contents'], 'test-pipeline.yaml', { type: 'text/yaml' });
      Object.defineProperty(uploadFile, 'files', {
        value: [file],
      });
      fireEvent.drop(uploadFile);

      const createBtn = await screen.findByText('Create');
      fireEvent.click(createBtn);

      await waitFor(() => {
        expect(uploadPipelineVersionSpy).toHaveBeenCalled();
      });

      expect(updateDialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'There was something wrong!',
          title: 'Pipeline version creation failed',
        }),
      );
    });
  });
});
