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

import * as React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import { ImportMethod, NewPipelineVersion } from './NewPipelineVersion';
import TestUtils from 'src/TestUtils';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';

class TestNewPipelineVersion extends NewPipelineVersion {
  public _onDropForTest = super._onDropForTest;
}

describe('NewPipelineVersion', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let pipelineVersionRef: React.RefObject<TestNewPipelineVersion> | null = null;

  const historyPushSpy = vi.fn();
  const historyReplaceSpy = vi.fn();
  const updateBannerSpy = vi.fn();
  const updateDialogSpy = vi.fn();
  const updateSnackbarSpy = vi.fn();
  const updateToolbarSpy = vi.fn();

  let getPipelineSpy: ReturnType<typeof vi.spyOn>;
  let createPipelineSpy: ReturnType<typeof vi.spyOn>;
  let createPipelineVersionSpy: ReturnType<typeof vi.spyOn>;
  let listPipelineVersionsSpy: ReturnType<typeof vi.spyOn>;
  let uploadPipelineSpy: ReturnType<typeof vi.spyOn>;

  const MOCK_PIPELINE = {
    pipeline_id: 'original-run-pipeline-id',
    display_name: 'original mock pipeline name',
  };

  const MOCK_PIPELINE_VERSION = {
    pipeline_version_id: 'original-run-pipeline-version-id',
    display_name: 'original mock pipeline version name',
    description: 'original mock pipeline version description',
    pipeline_id: 'original-run-pipeline-id',
  };

  function generateProps(search: string = ''): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_PIPELINE_VERSION,
        search,
      } as any,
      match: '' as any,
      toolbarProps: {} as any,
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    } as PageProps;
  }

  function getInstance(): TestNewPipelineVersion {
    if (!pipelineVersionRef?.current) {
      throw new Error('NewPipelineVersion instance not available');
    }
    return pipelineVersionRef.current;
  }

  async function renderNewPipelineVersion(
    search: string = '',
    propsPatch: Partial<PageProps & { buildInfo?: any; namespace?: string }> = {},
  ): Promise<void> {
    pipelineVersionRef = React.createRef<TestNewPipelineVersion>();
    const props = { ...generateProps(search), ...propsPatch } as PageProps;
    renderResult = render(<TestNewPipelineVersion ref={pipelineVersionRef} {...props} />);
    await TestUtils.flushPromises();
  }

  beforeEach(() => {
    vi.clearAllMocks();
    getPipelineSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'getPipeline')
      .mockResolvedValue(MOCK_PIPELINE as any);
    createPipelineVersionSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'createPipelineVersion')
      .mockResolvedValue(MOCK_PIPELINE_VERSION as any);
    createPipelineSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'createPipeline')
      .mockResolvedValue(MOCK_PIPELINE as any);
    listPipelineVersionsSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions')
      .mockResolvedValue({ pipeline_versions: [MOCK_PIPELINE_VERSION] } as any);
    uploadPipelineSpy = vi.spyOn(Apis, 'uploadPipelineV2').mockResolvedValue(MOCK_PIPELINE as any);
  });

  afterEach(async () => {
    renderResult?.unmount();
    renderResult = null;
    pipelineVersionRef = null;
    vi.restoreAllMocks();
  });

  // New pipeline version page has two functionalities: creating a pipeline and creating a version under an existing pipeline.
  // Our tests are divided into 3 parts: switching between creating pipeline or creating version; test pipeline creation; test pipeline version creation.

  describe('switching between creating pipeline and creating pipeline version', () => {
    it('creates pipeline is default when landing from pipeline list page', async () => {
      await renderNewPipelineVersion();

      // When landing from pipeline list page, the default is to create pipeline
      expect(getInstance().state.newPipeline).toBe(true);

      // Switch to create pipeline version
      fireEvent.click(
        screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
      );
      await waitFor(() => expect(getInstance().state.newPipeline).toBe(false));

      // Switch back
      fireEvent.click(screen.getByLabelText(/^Create a new pipeline$/));
      await waitFor(() => expect(getInstance().state.newPipeline).toBe(true));
    });

    it('creates pipeline version is default when landing from pipeline details page', async () => {
      await renderNewPipelineVersion(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`);

      // When landing from pipeline details page, the default is to create pipeline version
      expect(getInstance().state.newPipeline).toBe(false);

      // Switch to create pipeline
      fireEvent.click(screen.getByLabelText(/^Create a new pipeline$/));
      await waitFor(() => expect(getInstance().state.newPipeline).toBe(true));

      // Switch back
      fireEvent.click(
        screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
      );
      await waitFor(() => expect(getInstance().state.newPipeline).toBe(false));
    });
  });

  describe('creating version under an existing pipeline', () => {
    async function renderExistingPipeline(): Promise<void> {
      await renderNewPipelineVersion(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`);
      await waitFor(() => expect(getInstance().state.pipelineId).toBe(MOCK_PIPELINE.pipeline_id));
    }

    it('does not include any action buttons in the toolbar', async () => {
      await renderExistingPipeline();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [{ displayName: 'Pipeline Versions', href: '/pipeline_versions/new' }],
        pageTitle: 'New Pipeline',
      });
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating pipeline version name', async () => {
      await renderExistingPipeline();

      getInstance().handleChange('pipelineVersionName')({
        target: { value: 'version name' },
      });

      await waitFor(() =>
        expect(getInstance().state).toHaveProperty('pipelineVersionName', 'version name'),
      );
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating pipeline version description', async () => {
      await renderExistingPipeline();

      getInstance().handleChange('pipelineVersionDescription')({
        target: { value: 'some description' },
      });

      await waitFor(() =>
        expect(getInstance().state).toHaveProperty(
          'pipelineVersionDescription',
          'some description',
        ),
      );
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating package url', async () => {
      await renderExistingPipeline();

      getInstance().handleChange('packageUrl')({
        target: { value: 'https://dummy' },
      });

      await waitFor(() =>
        expect(getInstance().state).toHaveProperty('packageUrl', 'https://dummy'),
      );
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating code source', async () => {
      await renderExistingPipeline();

      getInstance().handleChange('codeSourceUrl')({
        target: { value: 'https://dummy' },
      });

      await waitFor(() =>
        expect(getInstance().state).toHaveProperty('codeSourceUrl', 'https://dummy'),
      );
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it("sends a request to create a version when 'Create' is clicked", async () => {
      await renderExistingPipeline();

      getInstance().handleChange('pipelineVersionName')({
        target: { value: 'test version name' },
      });
      getInstance().handleChange('pipelineVersionDisplayName')({
        target: { value: 'test version display name' },
      });
      getInstance().handleChange('pipelineVersionDescription')({
        target: { value: 'some description' },
      });
      getInstance().handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });

      await TestUtils.flushPromises();

      fireEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1));

      expect(createPipelineVersionSpy).toHaveBeenLastCalledWith('original-run-pipeline-id', {
        pipeline_id: 'original-run-pipeline-id',
        name: 'test version name',
        display_name: 'test version display name',
        description: 'some description',
        package_url: {
          pipeline_url: 'https://dummy_package_url',
        },
      });
    });
  });

  describe('creating new pipeline', () => {
    it('renders the new pipeline page', async () => {
      await renderNewPipelineVersion();
      await waitFor(() =>
        expect(
          screen.getByText(/Must specify either package url\s+or file in \.yaml/),
        ).toBeInTheDocument(),
      );
      expect(renderResult!.asFragment()).toMatchSnapshot();
    });

    it('switches between import methods', async () => {
      await renderNewPipelineVersion();

      // Import method is URL by default
      expect(getInstance().state.importMethod).toBe(ImportMethod.URL);

      // Click to import by local
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      await waitFor(() => expect(getInstance().state.importMethod).toBe(ImportMethod.LOCAL));

      // Click back to URL
      fireEvent.click(screen.getByLabelText(/Import by url/i));
      await waitFor(() => expect(getInstance().state.importMethod).toBe(ImportMethod.URL));
    });

    it('creates pipeline from url in single user mode', async () => {
      await renderNewPipelineVersion('', { buildInfo: { apiServerMultiUser: false } });

      getInstance().handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      getInstance().handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      getInstance().handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();

      fireEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));

      expect(getInstance().state).toHaveProperty('isPrivate', false);
      expect(getInstance().state).toHaveProperty('newPipeline', true);
      expect(getInstance().state).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
      });
    });

    it('creates private pipeline from url in multi user mode', async () => {
      await renderNewPipelineVersion('', {
        namespace: 'ns',
        buildInfo: { apiServerMultiUser: true },
      });

      getInstance().handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      getInstance().handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      getInstance().handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();
      fireEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));

      expect(getInstance().state).toHaveProperty('isPrivate', true);
      expect(getInstance().state).toHaveProperty('newPipeline', true);
      expect(getInstance().state).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
        namespace: 'ns',
      });
    });

    it('creates shared pipeline from url in multi user mode', async () => {
      await renderNewPipelineVersion('', {
        namespace: 'ns',
        buildInfo: { apiServerMultiUser: true },
      });

      getInstance().handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      getInstance().handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      getInstance().handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      getInstance().setState({ isPrivate: false });
      await TestUtils.flushPromises();
      fireEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));

      expect(getInstance().state).toHaveProperty('isPrivate', false);
      expect(getInstance().state).toHaveProperty('newPipeline', true);
      expect(getInstance().state).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
      });
    });

    it('creates pipeline from local file in single user mode', async () => {
      await renderNewPipelineVersion('', { buildInfo: { apiServerMultiUser: false } });

      // Set local file, pipeline name, pipeline description and click create
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      getInstance().handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      getInstance().handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      getInstance()._onDropForTest([file]);
      fireEvent.click(screen.getByRole('button', { name: 'Create' }));

      await TestUtils.flushPromises();

      expect(getInstance().state).toHaveProperty('isPrivate', false);
      expect(getInstance().state.importMethod).toBe(ImportMethod.LOCAL);

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        undefined,
      );
    });

    it('creates private pipeline from local file in multi user mode', async () => {
      await renderNewPipelineVersion('', {
        namespace: 'ns',
        buildInfo: { apiServerMultiUser: true },
      });

      // Set local file, pipeline name, pipeline description and click create
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      getInstance().handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      getInstance().handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      getInstance()._onDropForTest([file]);

      fireEvent.click(screen.getByRole('button', { name: 'Create' }));

      await TestUtils.flushPromises();

      expect(getInstance().state).toHaveProperty('isPrivate', true);
      expect(getInstance().state.importMethod).toBe(ImportMethod.LOCAL);

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        'ns',
      );
    });

    it('creates shared pipeline from local file in multi user mode', async () => {
      await renderNewPipelineVersion('', {
        namespace: 'ns',
        buildInfo: { apiServerMultiUser: true },
      });

      // Set local file, pipeline name, pipeline description and click create
      fireEvent.click(screen.getByLabelText(/Upload a file/i));
      getInstance().handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      getInstance().handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      getInstance()._onDropForTest([file]);
      getInstance().setState({ isPrivate: false });
      fireEvent.click(screen.getByRole('button', { name: 'Create' }));

      await TestUtils.flushPromises();

      expect(getInstance().state).toHaveProperty('isPrivate', false);
      expect(getInstance().state.importMethod).toBe(ImportMethod.LOCAL);

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        undefined,
      );
    });

    it('allows updating pipeline version name', async () => {
      await renderNewPipelineVersion(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`);
      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version Name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-pipeline-name' } });
      expect(pipelineVersionNameInput).toHaveValue('new-pipeline-name');
    });

    describe('kubernetes pipeline store', () => {
      beforeEach(() => {
        vi.spyOn(Apis.pipelineServiceApi, 'getPipeline').mockResolvedValue({
          name: 'test-pipeline',
        } as any);
      });

      it('shows pipeline display name field when pipeline store is kubernetes', async () => {
        await renderNewPipelineVersion('', { buildInfo: { pipelineStore: 'kubernetes' } });

        // Select the first radio button for "Create a new pipeline"
        const createNewPipelineBtn = screen.getByLabelText(/^Create a new pipeline$/);
        fireEvent.click(createNewPipelineBtn);

        // Pipeline display name field should be visible
        const pipelineDisplayNameInput = await screen.findByLabelText(/Pipeline Display Name/);
        expect(pipelineDisplayNameInput).toBeInTheDocument();

        // Pipeline name field should have Kubernetes object name label
        const pipelineNameInput = await screen.findByLabelText(
          /Pipeline Name \(Kubernetes object name\)/,
        );
        expect(pipelineNameInput).toBeInTheDocument();
      });

      it('shows pipeline version display name field when pipeline store is kubernetes', async () => {
        await renderNewPipelineVersion('', { buildInfo: { pipelineStore: 'kubernetes' } });

        // Select the second radio button for "Create a new pipeline version under an existing pipeline"
        const createVersionBtn = screen.getByLabelText(
          /^Create a new pipeline version under an existing pipeline$/,
        );
        fireEvent.click(createVersionBtn);

        // Pipeline version display name field should be visible
        const pipelineVersionDisplayNameInput = await screen.findByLabelText(
          /Pipeline Version Display Name/,
        );
        expect(pipelineVersionDisplayNameInput).toBeInTheDocument();

        // Pipeline version name field should have Kubernetes object name label
        const pipelineVersionNameInput = await screen.findByLabelText(
          /Pipeline Version Name \(Kubernetes object name\)/,
        );
        expect(pipelineVersionNameInput).toBeInTheDocument();
      });

      it('allows updating pipeline display name', async () => {
        await renderNewPipelineVersion('', { buildInfo: { pipelineStore: 'kubernetes' } });

        // Select the first radio button for "Create a new pipeline"
        const createNewPipelineBtn = screen.getByLabelText(/^Create a new pipeline$/);
        fireEvent.click(createNewPipelineBtn);

        // Update the display name
        const pipelineDisplayNameInput = await screen.findByLabelText(/Pipeline Display Name/);
        fireEvent.change(pipelineDisplayNameInput, {
          target: { value: 'Test Pipeline Display Name' },
        });
        expect(pipelineDisplayNameInput).toHaveValue('Test Pipeline Display Name');
      });

      it('allows updating pipeline version display name', async () => {
        await renderNewPipelineVersion('', { buildInfo: { pipelineStore: 'kubernetes' } });

        // Select the second radio button for "Create a new pipeline version under an existing pipeline"
        const createVersionBtn = screen.getByLabelText(
          /^Create a new pipeline version under an existing pipeline$/,
        );
        fireEvent.click(createVersionBtn);

        // Update the display name
        const pipelineVersionDisplayNameInput = await screen.findByLabelText(
          /Pipeline Version Display Name/,
        );
        fireEvent.change(pipelineVersionDisplayNameInput, {
          target: { value: 'Test Version Display Name' },
        });
        expect(pipelineVersionDisplayNameInput).toHaveValue('Test Version Display Name');
      });

      it('shows error message for invalid pipeline name', async () => {
        await renderNewPipelineVersion('', { buildInfo: { pipelineStore: 'kubernetes' } });

        // Select the first radio button for "Create a new pipeline"
        const createNewPipelineBtn = screen.getByLabelText(/^Create a new pipeline$/);
        fireEvent.click(createNewPipelineBtn);

        // Select "Import by url" and enter a mock URL
        const importByUrlBtn = screen.getByLabelText(/Import by url/);
        fireEvent.click(importByUrlBtn);
        const packageUrlInput = screen.getByLabelText(/Package Url/);
        fireEvent.change(packageUrlInput, {
          target: { value: 'https://example.com/pipeline.yaml' },
        });

        // Enter an invalid pipeline name (uppercase letters)
        const pipelineNameInput = await screen.findByLabelText(
          /Pipeline Name \(Kubernetes object name\)/,
        );
        fireEvent.change(pipelineNameInput, { target: { value: 'Invalid-Name' } });

        // Error message should be displayed
        const errorMessage = await screen.findByText(
          /Pipeline name must match Kubernetes naming pattern/,
        );
        expect(errorMessage).toBeInTheDocument();
      });
    });
  });
});
