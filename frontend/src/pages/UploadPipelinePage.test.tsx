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
import { fireEvent, render as renderWithoutTheme, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import { UploadPipelineController } from './UploadPipelinePage';
import { flushPromisesInAct, invokeAndFlush } from 'src/TestUtils';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';

function render(element: React.ReactElement) {
  return renderWithoutTheme(<ThemeProvider defaultTheme='light'>{element}</ThemeProvider>);
}
beforeEach(() => {
  localStorage.clear();
  vi.stubGlobal('matchMedia', (media: string) => ({
    matches: false,
    media,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});
afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

// Exposes protected drop handlers for testing (JSDOM lacks DataTransfer/FileList support).
class TestNewPipelineVersion extends UploadPipelineController {
  public simulateDrop(files: File[]): void {
    this._onDropForTest(files);
  }
  public simulateDropRejected(): void {
    (this as any)._onDropRejected();
  }
}

describe('NewPipelineVersion', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  // Ref is used ONLY for file drop simulation.
  // JSDOM lacks DataTransfer/FileList support needed by react-dropzone,
  // so we call the component's _onDrop handler directly for file upload tests.
  let componentRef: React.RefObject<TestNewPipelineVersion> | null = null;

  const navigateSpy = vi.fn();
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
      navigate: navigateSpy,
      location: {
        pathname: RoutePage.NEW_PIPELINE_VERSION,
        search,
      } as any,
      params: {},
      toolbarProps: {} as any,
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    } as PageProps;
  }

  async function renderNewPipelineVersion(
    search: string = '',
    propsPatch: Partial<PageProps & { buildInfo?: any; namespace?: string }> = {},
    { withRef = false } = {},
  ): Promise<void> {
    const props = { ...generateProps(search), ...propsPatch } as PageProps;
    if (withRef) {
      componentRef = React.createRef<TestNewPipelineVersion>();
      renderResult = render(<TestNewPipelineVersion ref={componentRef} {...props} />);
    } else {
      renderResult = render(<UploadPipelineController {...props} />);
    }
    await flushPromisesInAct();
  }

  /**
   * Simulate a file drop by calling the component's _onDrop handler directly.
   * This is necessary because JSDOM does not implement DataTransfer or FileList,
   * which react-dropzone requires for drop/input events to work.
   */
  async function simulateFileDrop(files: File[]): Promise<void> {
    if (!componentRef?.current) {
      throw new Error('componentRef not available — did you forget { withRef: true }?');
    }
    await invokeAndFlush(() => {
      componentRef!.current!.simulateDrop(files);
    });
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
    componentRef = null;
    vi.restoreAllMocks();
  });

  // New pipeline version page has two functionalities: creating a pipeline and creating a version under an existing pipeline.
  // Our tests are divided into 3 parts: switching between creating pipeline or creating version; test pipeline creation; test pipeline version creation.

  describe('switching between creating pipeline and creating pipeline version', () => {
    it('creates pipeline is default when landing from pipeline list page', async () => {
      await renderNewPipelineVersion();

      // When landing from pipeline list page, the default is to create pipeline
      expect(screen.getByLabelText(/^Create a new pipeline$/)).toBeChecked();

      // Switch to create pipeline version
      await userEvent.click(
        screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
      );
      await waitFor(() =>
        expect(
          screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
        ).toBeChecked(),
      );

      // Switch back
      await userEvent.click(screen.getByLabelText(/^Create a new pipeline$/));
      await waitFor(() => expect(screen.getByLabelText(/^Create a new pipeline$/)).toBeChecked());
    });

    it('creates pipeline version is default when landing from pipeline details page', async () => {
      await renderNewPipelineVersion(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`);

      // When landing from pipeline details page, the default is to create pipeline version
      expect(
        screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
      ).toBeChecked();

      // Switch to create pipeline
      await userEvent.click(screen.getByLabelText(/^Create a new pipeline$/));
      await waitFor(() => expect(screen.getByLabelText(/^Create a new pipeline$/)).toBeChecked());

      // Switch back
      await userEvent.click(
        screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
      );
      await waitFor(() =>
        expect(
          screen.getByLabelText(/^Create a new pipeline version under an existing pipeline$/),
        ).toBeChecked(),
      );
    });
  });

  describe('creating version under an existing pipeline', () => {
    async function renderExistingPipeline({ withRef = false } = {}): Promise<void> {
      await renderNewPipelineVersion(
        `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`,
        {},
        { withRef },
      );
      await waitFor(() => expect(getPipelineSpy).toHaveBeenCalledWith(MOCK_PIPELINE.pipeline_id));
    }

    it('does not include any action buttons in the toolbar', async () => {
      await renderExistingPipeline();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [{ displayName: 'Pipeline Versions', href: '/pipeline_versions/new' }],
        pageTitle: 'New Pipeline',
      });
      expect(getPipelineSpy).toHaveBeenCalledWith(MOCK_PIPELINE.pipeline_id);
    });

    it('allows updating pipeline version name', async () => {
      await renderExistingPipeline();

      const input = screen.getByLabelText(/Pipeline Version Name/);
      fireEvent.change(input, { target: { value: 'version name' } });

      await waitFor(() => expect(input).toHaveValue('version name'));
      expect(getPipelineSpy).toHaveBeenCalledWith(MOCK_PIPELINE.pipeline_id);
    });

    it('allows updating pipeline version description', async () => {
      await renderExistingPipeline();

      const input = screen.getByLabelText(/Pipeline Version Description/);
      fireEvent.change(input, { target: { value: 'some description' } });

      await waitFor(() => expect(input).toHaveValue('some description'));
      expect(getPipelineSpy).toHaveBeenCalledWith(MOCK_PIPELINE.pipeline_id);
    });

    it('allows updating package url', async () => {
      await renderExistingPipeline();

      const input = screen.getByLabelText(/Package Url/);
      fireEvent.change(input, { target: { value: 'https://dummy' } });

      await waitFor(() => expect(input).toHaveValue('https://dummy'));
      expect(getPipelineSpy).toHaveBeenCalledWith(MOCK_PIPELINE.pipeline_id);
    });

    it('allows updating code source', async () => {
      await renderExistingPipeline();

      const input = screen.getByLabelText(/Code Source/);
      fireEvent.change(input, { target: { value: 'https://dummy' } });

      await waitFor(() => expect(input).toHaveValue('https://dummy'));
      expect(getPipelineSpy).toHaveBeenCalledWith(MOCK_PIPELINE.pipeline_id);
    });

    it('preserves pipeline version name when a file drop is rejected', async () => {
      await renderExistingPipeline({ withRef: true });

      // Set a custom version name
      const versionNameInput = screen.getByLabelText(/Pipeline Version Name/);
      fireEvent.change(versionNameInput, { target: { value: 'my-custom-version-name' } });
      await waitFor(() => expect(versionNameInput).toHaveValue('my-custom-version-name'));

      // Simulate a rejected drop (JSDOM workaround — same as file drop tests)
      await invokeAndFlush(() => {
        componentRef!.current!.simulateDropRejected();
      });

      // Version name must NOT be cleared
      expect(versionNameInput).toHaveValue('my-custom-version-name');

      // Snackbar was shown to notify the user
      expect(updateSnackbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({ open: true, autoHideDuration: 5000 }),
      );
    });

    it("sends a request to create a version when 'Create' is clicked", async () => {
      await renderExistingPipeline();

      const versionNameInput = screen.getByLabelText(/Pipeline Version Name/);
      fireEvent.change(versionNameInput, { target: { value: 'test version name' } });

      const descInput = screen.getByLabelText(/Pipeline Version Description/);
      fireEvent.change(descInput, { target: { value: 'some description' } });

      const urlInput = screen.getByLabelText(/Package Url/);
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1));

      expect(createPipelineVersionSpy).toHaveBeenLastCalledWith('original-run-pipeline-id', {
        pipeline_id: 'original-run-pipeline-id',
        name: 'test version name',
        display_name: '',
        description: 'some description',
        package_url: {
          pipeline_url: 'https://dummy_package_url',
        },
      });
    });

    it('sends code_source_url when creating a version from file upload', async () => {
      const uploadVersionSpy = vi
        .spyOn(Apis, 'uploadPipelineVersionV2')
        .mockResolvedValue(MOCK_PIPELINE_VERSION as any);

      await renderExistingPipeline({ withRef: true });

      const versionNameInput = screen.getByLabelText(/Pipeline Version Name/);
      fireEvent.change(versionNameInput, { target: { value: 'test version name' } });

      await userEvent.click(screen.getByLabelText(/Upload a file/i));

      const file = new File(['file contents'], 'pipeline.yaml', { type: 'text/plain' });
      await simulateFileDrop([file]);

      const codeSourceInput = screen.getByLabelText(/Code Source/);
      fireEvent.change(codeSourceInput, { target: { value: 'https://github.com/example/repo' } });

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(uploadVersionSpy).toHaveBeenCalledTimes(1));

      expect(uploadVersionSpy).toHaveBeenLastCalledWith(
        'test version name',
        '',
        'original-run-pipeline-id',
        file,
        '',
        'https://github.com/example/repo',
      );
    });

    it('sends code_source_url when creating a version from URL with code source filled', async () => {
      await renderExistingPipeline();

      const versionNameInput = screen.getByLabelText(/Pipeline Version Name/);
      fireEvent.change(versionNameInput, { target: { value: 'test version name' } });

      const urlInput = screen.getByLabelText(/Package Url/);
      fireEvent.change(urlInput, { target: { value: 'https://dummy_package_url' } });

      const codeSourceInput = screen.getByLabelText(/Code Source/);
      fireEvent.change(codeSourceInput, { target: { value: 'https://github.com/example/repo' } });

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1));

      expect(createPipelineVersionSpy).toHaveBeenLastCalledWith('original-run-pipeline-id', {
        pipeline_id: 'original-run-pipeline-id',
        name: 'test version name',
        display_name: '',
        description: '',
        package_url: {
          pipeline_url: 'https://dummy_package_url',
        },
        code_source_url: 'https://github.com/example/repo',
      });
    });
  });

  describe('creating new pipeline', () => {
    it('renders the new pipeline page', async () => {
      await renderNewPipelineVersion();
      expect(
        screen.getByText(/Must specify either package url\s+or file in \.yaml/),
      ).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled();
    });

    it('switches between import methods', async () => {
      await renderNewPipelineVersion();

      // Import method is URL by default
      expect(screen.getByLabelText(/Import by url/i)).toBeChecked();

      // Click to import by local
      await userEvent.click(screen.getByLabelText(/Upload a file/i));
      await waitFor(() => expect(screen.getByLabelText(/Upload a file/i)).toBeChecked());

      // Click back to URL
      await userEvent.click(screen.getByLabelText(/Import by url/i));
      await waitFor(() => expect(screen.getByLabelText(/Import by url/i)).toBeChecked());
    });

    it('creates pipeline from url in single user mode', async () => {
      await renderNewPipelineVersion('', { buildInfo: { apiServerMultiUser: false } });

      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));

      expect(screen.getByLabelText(/^Create a new pipeline$/)).toBeChecked();
      expect(screen.getByLabelText(/Import by url/i)).toBeChecked();
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
      });
    });

    it('submits a valid form with Enter and blocks duplicate submits and cancel while pending', async () => {
      let resolveCreate!: (value: typeof MOCK_PIPELINE) => void;
      createPipelineSpy.mockReturnValue(
        new Promise((resolve) => {
          resolveCreate = resolve;
        }),
      );
      await renderNewPipelineVersion();
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'keyboard pipeline' },
      });
      const packageUrl = screen.getByLabelText(/Package Url/);
      fireEvent.change(packageUrl, { target: { value: 'https://example.com/pipeline.yaml' } });
      await userEvent.click(screen.getByLabelText(/Pipeline Name/));
      await userEvent.keyboard('{Enter}');
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));
      expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled();
      expect(screen.getByRole('button', { name: 'Create' })).toHaveAttribute('aria-busy', 'true');
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
      fireEvent.submit(packageUrl.closest('form')!);
      fireEvent.submit(packageUrl.closest('form')!);
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(navigateSpy).not.toHaveBeenCalled();
      await invokeAndFlush(() => resolveCreate(MOCK_PIPELINE));
      await waitFor(() => expect(createPipelineVersionSpy).toHaveBeenCalledOnce());
      expect(navigateSpy).toHaveBeenCalledOnce();
    });

    it('does not submit incomplete forms or interpret Escape as page cancellation', async () => {
      await renderNewPipelineVersion();
      await userEvent.click(screen.getByLabelText(/Pipeline Name/));
      await userEvent.keyboard('draft{Enter}{Escape}');
      expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled();
      expect(screen.getByLabelText(/Pipeline Name/)).toHaveValue('draft');
      expect(createPipelineSpy).not.toHaveBeenCalled();
      expect(navigateSpy).not.toHaveBeenCalled();
      await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      expect(navigateSpy).toHaveBeenCalledWith(RoutePage.PIPELINES);
      expect(createPipelineSpy).not.toHaveBeenCalled();
    });

    it('keeps Enter in multiline fields as a newline rather than submitting', async () => {
      await renderNewPipelineVersion();
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), { target: { value: 'pipeline' } });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://example.com/pipeline.yaml' },
      });
      const source = screen.getByLabelText('Code Source');
      await userEvent.type(source, 'first{Enter}second');
      expect(source).toHaveValue('first\nsecond');
      expect(createPipelineSpy).not.toHaveBeenCalled();
    });

    it('retains the draft and allows retry after creation fails', async () => {
      const log = vi.spyOn(console, 'error').mockImplementation(() => {});
      createPipelineSpy.mockRejectedValueOnce(new Error('Creation unavailable'));
      await renderNewPipelineVersion();
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'retry pipeline' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://example.com/pipeline.yaml' },
      });
      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(screen.getByRole('button', { name: 'Create' })).toBeEnabled());
      expect(screen.getByLabelText(/Pipeline Name/)).toHaveValue('retry pipeline');
      expect(updateDialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({ title: 'Pipeline version creation failed' }),
      );
      expect(navigateSpy).not.toHaveBeenCalled();
      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(navigateSpy).toHaveBeenCalledOnce());
      expect(createPipelineSpy).toHaveBeenCalledTimes(2);
      log.mockRestore();
    });

    it('uses the newly created pipeline ID for its URL-imported version', async () => {
      const createdId = 'new/pipeline%id';
      createPipelineSpy.mockResolvedValue({ pipeline_id: createdId });
      createPipelineVersionSpy.mockResolvedValue({
        ...MOCK_PIPELINE_VERSION,
        pipeline_id: createdId,
      });
      await renderNewPipelineVersion('', { buildInfo: { apiServerMultiUser: false } });
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'new pipeline' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://example.test/pipeline.yaml' },
      });

      await userEvent.click(screen.getByRole('button', { name: 'Create', exact: true }));
      await waitFor(() => expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1));
      expect(createPipelineVersionSpy).toHaveBeenCalledWith(
        createdId,
        expect.objectContaining({
          pipeline_id: createdId,
          package_url: { pipeline_url: 'https://example.test/pipeline.yaml' },
        }),
      );
      await waitFor(() => expect(navigateSpy).toHaveBeenCalledTimes(1));
      expect(new URL(navigateSpy.mock.calls[0][0], 'http://kfp.test').pathname).toBe(
        `/pipelines/details/${encodeURIComponent(createdId)}/version/original-run-pipeline-version-id`,
      );
    });

    it('creates private pipeline from url in multi user mode', async () => {
      await renderNewPipelineVersion('', {
        namespace: 'ns',
        buildInfo: { apiServerMultiUser: true },
      });

      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });
      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));

      expect(screen.getByLabelText(/^Create a new pipeline$/)).toBeChecked();
      expect(screen.getByLabelText(/Import by url/i)).toBeChecked();
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

      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });
      fireEvent.change(screen.getByLabelText(/Package Url/), {
        target: { value: 'https://dummy_package_url' },
      });

      // Click "Shared" to set isPrivate to false
      const sharedRadio = screen.getByLabelText(/Shared/i);
      await userEvent.click(sharedRadio);

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(createPipelineSpy).toHaveBeenCalledTimes(1));

      expect(screen.getByLabelText(/^Create a new pipeline$/)).toBeChecked();
      expect(screen.getByLabelText(/Import by url/i)).toBeChecked();
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        display_name: 'test pipeline name',
      });
    }, 20000);

    it('creates pipeline from local file in single user mode', async () => {
      await renderNewPipelineVersion(
        '',
        { buildInfo: { apiServerMultiUser: false } },
        { withRef: true },
      );

      await userEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });

      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      await simulateFileDrop([file]);

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));

      await waitFor(() => expect(uploadPipelineSpy).toHaveBeenCalled());

      expect(screen.getByLabelText(/Upload a file/i)).toBeChecked();

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        undefined,
        undefined,
      );
    });

    it('creates private pipeline from local file in multi user mode', async () => {
      await renderNewPipelineVersion(
        '',
        {
          namespace: 'ns',
          buildInfo: { apiServerMultiUser: true },
        },
        { withRef: true },
      );

      await userEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });

      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      await simulateFileDrop([file]);

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));

      await waitFor(() => expect(uploadPipelineSpy).toHaveBeenCalled());

      expect(screen.getByLabelText(/Upload a file/i)).toBeChecked();

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        'ns',
        undefined,
      );
    });

    it('creates shared pipeline from local file in multi user mode', async () => {
      await renderNewPipelineVersion(
        '',
        {
          namespace: 'ns',
          buildInfo: { apiServerMultiUser: true },
        },
        { withRef: true },
      );

      await userEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });

      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      await simulateFileDrop([file]);

      // Click "Shared" to set isPrivate to false
      const sharedRadio = screen.getByLabelText(/Shared/i);
      await userEvent.click(sharedRadio);

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));

      await waitFor(() => expect(uploadPipelineSpy).toHaveBeenCalled());

      expect(screen.getByLabelText(/Upload a file/i)).toBeChecked();

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        undefined,
        undefined,
      );
    });

    it('creates pipeline from local file with code source url', async () => {
      await renderNewPipelineVersion(
        '',
        { buildInfo: { apiServerMultiUser: false } },
        { withRef: true },
      );

      await userEvent.click(screen.getByLabelText(/Upload a file/i));
      fireEvent.change(screen.getByLabelText(/Pipeline Name/), {
        target: { value: 'test pipeline name' },
      });
      fireEvent.change(screen.getByLabelText(/Pipeline Description/), {
        target: { value: 'test pipeline description' },
      });

      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      await simulateFileDrop([file]);

      fireEvent.change(screen.getByLabelText(/Code Source/), {
        target: { value: 'https://github.com/example/repo' },
      });

      await userEvent.click(screen.getByRole('button', { name: 'Create' }));
      await waitFor(() => expect(uploadPipelineSpy).toHaveBeenCalled());

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        '',
        'test pipeline description',
        file,
        undefined,
        'https://github.com/example/repo',
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
        vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline').mockResolvedValue({
          name: 'test-pipeline',
        } as any);
      });

      it('shows pipeline display name field when pipeline store is kubernetes', async () => {
        await renderNewPipelineVersion('', { buildInfo: { pipelineStore: 'kubernetes' } });

        // Select the first radio button for "Create a new pipeline"
        const createNewPipelineBtn = screen.getByLabelText(/^Create a new pipeline$/);
        await userEvent.click(createNewPipelineBtn);

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
        await userEvent.click(createVersionBtn);

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
        await userEvent.click(createNewPipelineBtn);

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
        await userEvent.click(createVersionBtn);

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
        await userEvent.click(createNewPipelineBtn);

        // Select "Import by url" and enter a mock URL
        const importByUrlBtn = screen.getByLabelText(/Import by url/);
        await userEvent.click(importByUrlBtn);
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
