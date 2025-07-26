/*
 * Copyright 2019 The Kubeflow Authors
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
import NewPipelineVersion, { NewPipelineVersionProps } from './NewPipelineVersion';
import TestUtils from 'src/TestUtils';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';
import '@testing-library/jest-dom';
import userEvent from '@testing-library/user-event';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { NamespaceContext } from 'src/lib/KubeflowClient';

// Mock missing renderer
(global as any).descriptionCustomRenderer = jest.fn();

describe('NewPipelineVersion', () => {
  const navigateSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  let getPipelineSpy: jest.SpyInstance<{}>;
  let createPipelineSpy: jest.SpyInstance<{}>;
  let createPipelineVersionSpy: jest.SpyInstance<{}>;
  let uploadPipelineSpy: jest.SpyInstance<{}>;

  let MOCK_PIPELINE = {
    pipeline_id: 'original-run-pipeline-id',
    display_name: 'original mock pipeline name',
  };

  let MOCK_PIPELINE_VERSION = {
    pipeline_version_id: 'original-run-pipeline-version-id',
    display_name: 'original mock pipeline version name',
    description: 'original mock pipeline version description',
  };

  function generateProps(search?: string): PageProps & NewPipelineVersionProps {
    return {
      navigate: navigateSpy,
      location: {
        pathname: RoutePage.NEW_PIPELINE_VERSION,
        search: search || '',
        hash: '',
        state: null,
        key: 'default',
      },
      match: { params: {}, isExact: true, path: '', url: '' },
      toolbarProps: {} as any,
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
      buildInfo: { pipelineStore: 'kubernetes' },
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
    navigateSpy.mockClear();

    // Reset all spies to ensure clean state
    if (getPipelineSpy) getPipelineSpy.mockRestore();
    if (createPipelineVersionSpy) createPipelineVersionSpy.mockRestore();
    if (createPipelineSpy) createPipelineSpy.mockRestore();
    if (uploadPipelineSpy) uploadPipelineSpy.mockRestore();

    getPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApiV2, 'getPipeline')
      .mockResolvedValue(MOCK_PIPELINE);
    createPipelineVersionSpy = jest
      .spyOn(Apis.pipelineServiceApiV2, 'createPipelineVersion')
      .mockResolvedValue(MOCK_PIPELINE_VERSION);
    createPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApiV2, 'createPipeline')
      .mockResolvedValue(MOCK_PIPELINE);
    uploadPipelineSpy = jest.spyOn(Apis, 'uploadPipelineV2').mockResolvedValue(MOCK_PIPELINE);
  });

  afterEach(async () => {
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  describe('switching between creating pipeline and creating pipeline version', () => {
    it('creates pipeline is default when landing from pipeline list page', () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />);
      const createPipelineBtn = screen.getByTestId('createNewPipelineBtn').querySelector('input');
      if (!createPipelineBtn) {
        throw new Error('createPipelineBtn not found');
      }
      expect(createPipelineBtn).toBeChecked();
      const createPipelineVersionBtn = screen.getByLabelText(
        /Create a new pipeline version under an existing pipeline/i,
      );
      fireEvent.click(createPipelineVersionBtn);
      expect(createPipelineVersionBtn).toBeChecked();
      expect(createPipelineBtn).not.toBeChecked();
      fireEvent.click(createPipelineBtn);
      expect(createPipelineBtn).toBeChecked();
      expect(createPipelineVersionBtn).not.toBeChecked();
    });

    it('creates pipeline version is default when landing from pipeline details page', () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      const createPipelineVersionBtn = screen.getByLabelText(
        /Create a new pipeline version under an existing pipeline/i,
      );
      expect(createPipelineVersionBtn).toBeChecked();
      const createPipelineBtn = screen.getByTestId('createNewPipelineBtn').querySelector('input');
      if (!createPipelineBtn) {
        throw new Error('createPipelineBtn not found');
      }
      fireEvent.click(createPipelineBtn);
      expect(createPipelineBtn).toBeChecked();
      expect(createPipelineVersionBtn).not.toBeChecked();
      fireEvent.click(createPipelineVersionBtn);
      expect(createPipelineVersionBtn).toBeChecked();
      expect(createPipelineBtn).not.toBeChecked();
    });
  });

  describe('creating version under an existing pipeline', () => {
    it('does not include any action buttons in the toolbar', async () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      await TestUtils.flushPromises();
      expect(getPipelineSpy).toHaveBeenCalledTimes(2);

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [{ displayName: 'Pipeline Versions', href: '/pipeline_versions/new' }],
        pageTitle: 'New Pipeline',
      });
    });

    it('allows updating pipeline version name', async () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      const input = await screen.findByLabelText(/Pipeline Version Name/);
      fireEvent.change(input, { target: { value: 'version name' } });
      expect(input).toHaveValue('version name');
      expect(getPipelineSpy).toHaveBeenCalledTimes(3);
    });

    it('allows updating pipeline version description', async () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      const input = await screen.findByLabelText(/Pipeline Version Description/);
      fireEvent.change(input, { target: { value: 'some description' } });
      expect(input).toHaveValue('some description');
      expect(getPipelineSpy).toHaveBeenCalledTimes(3);
    });

    it('allows updating package url', async () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      const input = await screen.findByLabelText(/Package Url/);
      fireEvent.change(input, { target: { value: 'https://dummy' } });
      expect(input).toHaveValue('https://dummy');
      expect(getPipelineSpy).toHaveBeenCalledTimes(3);
    });

    it('allows updating code source', async () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      const input = await screen.findByLabelText(/Code Source/);
      fireEvent.change(input, { target: { value: 'https://dummy' } });
      expect(input).toHaveValue('https://dummy');
      expect(getPipelineSpy).toHaveBeenCalledTimes(3);
    });

    it.skip("sends a request to create a version when 'Create' is clicked", async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ pipelineStore: 'kubernetes' }}>
          <NewPipelineVersion {...generateProps()} />
        </BuildInfoContext.Provider>,
        [`/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`],
      );
      fireEvent.change(await screen.findByLabelText(/Pipeline Version Name/), {
        target: { value: 'test-version-name' },
      });
      fireEvent.change(await screen.findByLabelText(/Pipeline Version Display Name/), {
        target: { value: 'test version display name' },
      });
      fireEvent.change(await screen.findByLabelText(/Pipeline Version Description/), {
        target: { value: 'some description' },
      });
      fireEvent.change(await screen.findByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();
      expect(screen.getByRole('button', { name: /create/i })).toBeEnabled();
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineVersionSpy).toHaveBeenLastCalledWith('original-run-pipeline-id', {
        pipeline_id: 'original-run-pipeline-id',
        name: 'test-version-name',
        display_name: 'test version display name',
        description: 'some description',
        package_url: {
          pipeline_url: 'https://dummy_package_url',
        },
      });
    });
  });

  describe('creating new pipeline', () => {
    it.skip('renders the new pipeline page', async () => {
      const { asFragment } = TestUtils.renderWithRouter(
        <NewPipelineVersion {...generateProps()} />,
      );
      await TestUtils.flushPromises();
      expect(asFragment()).toMatchSnapshot();
    });

    it('switches between import methods', () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />);
      const remoteBtn = screen.getByLabelText(/Import by url/i);
      const localBtn = screen.getByLabelText(/Upload a file/i);
      expect(remoteBtn).toBeChecked();
      fireEvent.click(localBtn);
      expect(localBtn).toBeChecked();
      expect(remoteBtn).not.toBeChecked();
      fireEvent.click(remoteBtn);
      expect(remoteBtn).toBeChecked();
      expect(localBtn).not.toBeChecked();
    });

    it('creates pipeline from url in single user mode', async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
          <NewPipelineVersion {...generateProps()} />
        </BuildInfoContext.Provider>,
        ['/does-not-matter'],
      );
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
      });
    });

    it('creates private pipeline from url in multi user mode', async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NamespaceContext.Provider value='ns'>
            <NewPipelineVersion {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>,
      );
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
        namespace: 'ns',
      });
    });

    it('creates shared pipeline from url in multi user mode', async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NewPipelineVersion {...generateProps()} />
        </BuildInfoContext.Provider>,
      );
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });
      // Simulate shared pipeline by toggling the isPrivate checkbox if present
      const isPrivateCheckbox = screen.queryByLabelText(/Private/);
      if (isPrivateCheckbox) {
        fireEvent.click(isPrivateCheckbox);
      }
      await TestUtils.flushPromises();
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
        namespace: undefined,
      });
    });

    // TODO: File upload not working in this test
    it.skip('creates pipeline from local file in single user mode', async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
          <NewPipelineVersion {...generateProps()} />
        </BuildInfoContext.Provider>,
      );
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test-pipeline-name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      // Simulate file drop
      // @ts-ignore
      screen.getByTestId('uploadFileInput').files = [file];
      fireEvent.change(screen.getByTestId('uploadFileInput'));
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test-pipeline-name',
        '',
        'test pipeline description',
        file,
        undefined,
      );
    });
    // TODO: File upload not working in this test
    it.skip('creates private pipeline from local file in multi user mode', async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NewPipelineVersion {...generateProps()} />
        </BuildInfoContext.Provider>,
      );
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test-pipeline-name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      // @ts-ignore
      screen.getByTestId('uploadFileInput').files = [file];
      fireEvent.change(screen.getByTestId('uploadFileInput'));
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test-pipeline-name',
        '',
        'test pipeline description',
        file,
        'ns',
      );
    });

    // TODO: File upload not working in this test
    it.skip('creates shared pipeline from local file in multi user mode', async () => {
      TestUtils.renderWithRouter(
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NewPipelineVersion {...generateProps()} />
        </BuildInfoContext.Provider>,
      );
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test-pipeline-name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      const dropzone = screen.getByTestId('uploadFileInput');
      const fileInput = dropzone.querySelector('input[type="file"]');
      if (!fileInput) {
        throw new Error('fileInput not found');
      }
      fileInput.removeAttribute('disabled');
      userEvent.upload(fileInput as HTMLInputElement, file);
      // Simulate shared pipeline by toggling the isPrivate checkbox if present
      const isPrivateCheckbox = screen.queryByLabelText(/Private/);
      if (isPrivateCheckbox) {
        fireEvent.click(isPrivateCheckbox);
      }
      fireEvent.click(screen.getByRole('button', { name: /create/i }));
      await TestUtils.flushPromises();
      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test-pipeline-name',
        '',
        'test pipeline description',
        file,
        undefined,
      );
    });

    it('allows updating pipeline version name', async () => {
      TestUtils.renderWithRouter(<NewPipelineVersion {...generateProps()} />, [
        `/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
      ]);
      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version Name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-pipeline-name' } });
      expect(pipelineVersionNameInput.closest('input')?.value).toBe('new-pipeline-name');
    });

    describe('kubernetes pipeline store', () => {
      beforeEach(() => {
        jest.spyOn(Apis.pipelineServiceApi, 'getPipeline').mockResolvedValue({
          name: 'test-pipeline',
        });
      });

      it('shows pipeline display name field when pipeline store is kubernetes', async () => {
        TestUtils.renderWithRouter(
          <BuildInfoContext.Provider value={{ pipelineStore: 'kubernetes' }}>
            <NewPipelineVersion {...generateProps()} />
          </BuildInfoContext.Provider>,
        );
        const createNewPipelineBtns = screen.getAllByLabelText(/Create a new pipeline/);
        fireEvent.click(createNewPipelineBtns[0]);
        const pipelineDisplayNameInput = await screen.findByLabelText(/Pipeline Display Name/);
        expect(pipelineDisplayNameInput).toBeInTheDocument();
        const pipelineNameInput = await screen.findByLabelText(
          /Pipeline Name \(Kubernetes object name\)/,
        );
        expect(pipelineNameInput).toBeInTheDocument();
      });

      it('shows pipeline version display name field when pipeline store is kubernetes', async () => {
        TestUtils.renderWithRouter(
          <BuildInfoContext.Provider value={{ pipelineStore: 'kubernetes' }}>
            <NewPipelineVersion {...generateProps()} />
          </BuildInfoContext.Provider>,
          [`/pipeline_versions/new?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`],
        );
        const pipelineVersionDisplayNameInput = await screen.findByLabelText(
          /Pipeline Version Display Name/,
        );
        expect(pipelineVersionDisplayNameInput).toBeInTheDocument();
        const pipelineVersionNameInput = await screen.findByLabelText(
          /Pipeline Version Name \(Kubernetes object name\)/,
        );
        expect(pipelineVersionNameInput).toBeInTheDocument();
      });

      it('allows updating pipeline display name', async () => {
        TestUtils.renderWithRouter(
          <BuildInfoContext.Provider value={{ pipelineStore: 'kubernetes' }}>
            <NewPipelineVersion {...generateProps()} />
          </BuildInfoContext.Provider>,
        );
        const createNewPipelineBtns = screen.getAllByLabelText(/Create a new pipeline/);
        fireEvent.click(createNewPipelineBtns[0]);
        const pipelineDisplayNameInput = await screen.findByLabelText(/Pipeline Display Name/);
        fireEvent.change(pipelineDisplayNameInput, {
          target: { value: 'Test Pipeline Display Name' },
        });
        expect(pipelineDisplayNameInput).toHaveValue('Test Pipeline Display Name');
      });

      it('allows updating pipeline version display name', async () => {
        TestUtils.renderWithRouter(
          <BuildInfoContext.Provider value={{ pipelineStore: 'kubernetes' }}>
            <NewPipelineVersion {...generateProps()} />
          </BuildInfoContext.Provider>,
        );
        const createVersionBtns = screen.getAllByLabelText(
          /Create a new pipeline version under an existing pipeline/,
        );
        fireEvent.click(createVersionBtns[0]);
        const pipelineVersionDisplayNameInput = await screen.findByLabelText(
          /Pipeline Version Display Name/,
        );
        fireEvent.change(pipelineVersionDisplayNameInput, {
          target: { value: 'Test Version Display Name' },
        });
        expect(pipelineVersionDisplayNameInput).toHaveValue('Test Version Display Name');
      });

      it('shows error message for invalid pipeline name', async () => {
        TestUtils.renderWithRouter(
          <BuildInfoContext.Provider value={{ pipelineStore: 'kubernetes' }}>
            <NewPipelineVersion {...generateProps()} />
          </BuildInfoContext.Provider>,
        );
        const createNewPipelineBtns = screen.getAllByLabelText(/Create a new pipeline/);
        fireEvent.click(createNewPipelineBtns[0]);
        const importByUrlBtn = screen.getByLabelText(/Import by url/);
        fireEvent.click(importByUrlBtn);
        const packageUrlInput = screen.getByLabelText(/Package Url/);
        fireEvent.change(packageUrlInput, {
          target: { value: 'https://example.com/pipeline.yaml' },
        });
        const pipelineNameInput = await screen.findByLabelText(
          /Pipeline Name \(Kubernetes object name\)/,
        );
        fireEvent.change(pipelineNameInput, { target: { value: 'Invalid-Name' } });
        const errorMessage = await screen.findByText(
          /Pipeline name must match Kubernetes naming pattern/,
        );
        expect(errorMessage).toBeInTheDocument();
      });
    });
  });
});
