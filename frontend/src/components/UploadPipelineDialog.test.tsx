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
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import UploadPipelineDialog, {
  ImportMethod,
  PIPELINE_PACKAGE_ACCEPT,
  PIPELINE_PACKAGE_REJECT_MESSAGE,
  pipelinePackageValidator,
} from './UploadPipelineDialog';

describe('PIPELINE_PACKAGE_ACCEPT', () => {
  it('accepts only backend-supported formats', () => {
    const allExtensions = Object.values(PIPELINE_PACKAGE_ACCEPT).flat().sort();
    expect(allExtensions).toEqual(['.tar.gz', '.yaml', '.yml', '.zip']);
  });

  it('does not include plain .gz (backend only supports .tar.gz)', () => {
    const allExtensions = Object.values(PIPELINE_PACKAGE_ACCEPT).flat();
    const plainGz = allExtensions.filter((ext) => ext.endsWith('.gz') && ext !== '.tar.gz');
    expect(plainGz).toEqual([]);
  });

  it('uses application/gzip MIME key for .tar.gz so native file picker works', () => {
    expect(PIPELINE_PACKAGE_ACCEPT).toHaveProperty('application/gzip');
    expect(PIPELINE_PACKAGE_ACCEPT['application/gzip']).toEqual(['.tar.gz']);
  });

  it('reject message matches accepted extensions and does not advertise .gz', () => {
    expect(PIPELINE_PACKAGE_REJECT_MESSAGE).not.toContain('.gz,');
    expect(PIPELINE_PACKAGE_REJECT_MESSAGE).toContain('.tar.gz');
    expect(PIPELINE_PACKAGE_REJECT_MESSAGE).toContain('.yaml');
    expect(PIPELINE_PACKAGE_REJECT_MESSAGE).toContain('.yml');
    expect(PIPELINE_PACKAGE_REJECT_MESSAGE).toContain('.zip');
  });
});

describe('pipelinePackageValidator', () => {
  it('accepts .tar.gz files', () => {
    expect(pipelinePackageValidator(new File([], 'pipeline.tar.gz'))).toBeNull();
  });

  it('accepts .tar.gz files with uppercase names', () => {
    expect(pipelinePackageValidator(new File([], 'PIPELINE.TAR.GZ'))).toBeNull();
  });

  it('accepts non-gzip files (validator only guards gzip MIME loophole)', () => {
    expect(pipelinePackageValidator(new File([], 'pipeline.yaml'))).toBeNull();
    expect(pipelinePackageValidator(new File([], 'pipeline.zip'))).toBeNull();
  });

  it('rejects plain .gz files', () => {
    expect(pipelinePackageValidator(new File([], 'foo.gz'))).toEqual(
      expect.objectContaining({ code: 'invalid-extension' }),
    );
    expect(pipelinePackageValidator(new File([], 'pipeline.yaml.gz'))).toEqual(
      expect.objectContaining({ code: 'invalid-extension' }),
    );
  });

  it('rejects .tgz files', () => {
    expect(pipelinePackageValidator(new File([], 'pipeline.tgz'))).toEqual(
      expect.objectContaining({ code: 'invalid-extension' }),
    );
  });
});

describe('UploadPipelineDialog', () => {
  const user = userEvent.setup();

  function renderDialog(onClose = vi.fn().mockResolvedValue(false)) {
    render(<UploadPipelineDialog open={true} onClose={onClose} />);
    return { onClose };
  }

  async function dropFile(file: File): Promise<void> {
    const dropzone = screen.getByTestId('upload-pipeline-dropzone');
    // react-dropzone hides the file input with no accessible label; direct query required.
    const input = dropzone.querySelector('input') as HTMLInputElement;
    await user.upload(input, file);
    await waitFor(() => {
      expect(screen.getByRole('textbox', { name: /Pipeline name/i })).not.toHaveValue('');
    });
  }

  it('shows dialog title when open', () => {
    renderDialog();
    expect(
      screen.getByRole('heading', { name: /upload and name your pipeline/i }),
    ).toBeInTheDocument();
  });

  it('does not render dialog content when closed', () => {
    render(<UploadPipelineDialog open={false} onClose={vi.fn().mockResolvedValue(false)} />);
    expect(
      screen.queryByRole('heading', { name: /upload and name your pipeline/i }),
    ).not.toBeInTheDocument();
  });

  it('populates pipeline name from filename when a file is dropped', async () => {
    renderDialog();
    const file = new File(['test'], 'my_pipeline.yaml', { type: 'application/yaml' });
    await dropFile(file);
    expect(screen.getByRole('textbox', { name: /Pipeline name/i })).toHaveValue('my_pipeline');
    expect(screen.queryByText('Drop files..')).not.toBeInTheDocument();
  });

  it('trims compound extension (.tar.gz) for pipeline name suggestion', async () => {
    renderDialog();
    const file = new File(['test'], 'test_upload_file.tar.gz', { type: 'application/gzip' });
    await dropFile(file);
    expect(screen.getByRole('textbox', { name: /Pipeline name/i })).toHaveValue('test_upload_file');
  });

  it('switches between local upload and URL import modes', async () => {
    renderDialog();
    expect(screen.getByTestId('upload-pipeline-dropzone')).toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: /URL/i })).not.toBeInTheDocument();

    await user.click(screen.getByLabelText('Import by URL'));
    expect(screen.queryByTestId('upload-pipeline-dropzone')).not.toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /URL/i })).toBeInTheDocument();

    await user.click(screen.getByLabelText('Upload a file'));
    expect(screen.getByTestId('upload-pipeline-dropzone')).toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: /URL/i })).not.toBeInTheDocument();
  });

  it('calls onClose with defaults when Cancel is clicked', async () => {
    const { onClose } = renderDialog();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledWith(false, '', null, '', ImportMethod.LOCAL, true, '');
  });

  it('calls onClose with file and pipeline name when Upload is clicked', async () => {
    const { onClose } = renderDialog();
    const file = new File(['test'], 'my_pipeline.yaml', { type: 'application/yaml' });
    await dropFile(file);

    const nameInput = screen.getByRole('textbox', { name: /Pipeline name/i });
    await user.clear(nameInput);
    await user.type(nameInput, 'test name');

    await user.click(screen.getByRole('button', { name: 'Upload' }));
    expect(onClose).toHaveBeenLastCalledWith(
      true,
      'test name',
      expect.any(File),
      '',
      ImportMethod.LOCAL,
      true,
      '',
    );
  });

  it('calls onClose with trimmed URL when Upload is clicked in URL mode', async () => {
    const { onClose } = renderDialog();
    await user.click(screen.getByLabelText('Import by URL'));

    const urlInput = screen.getByRole('textbox', { name: /URL/i });
    await user.type(urlInput, '  https://www.google.com/test-file.txt  ');

    const nameInput = screen.getByRole('textbox', { name: /Pipeline name/i });
    await user.type(nameInput, 'test name');

    await user.click(screen.getByRole('button', { name: 'Upload' }));
    expect(onClose).toHaveBeenLastCalledWith(
      true,
      'test name',
      null,
      'https://www.google.com/test-file.txt',
      ImportMethod.URL,
      true,
      '',
    );
  });

  it('resets all inputs after a successful upload', async () => {
    const onClose = vi.fn().mockResolvedValue(true);
    renderDialog(onClose);
    const file = new File(['test'], 'my_pipeline.yaml', { type: 'application/yaml' });
    await dropFile(file);

    const nameInput = screen.getByRole('textbox', { name: /Pipeline name/i });
    await user.clear(nameInput);
    await user.type(nameInput, 'custom name');

    await user.click(screen.getByRole('button', { name: 'Upload' }));
    await waitFor(() => {
      expect(screen.getByRole('textbox', { name: /Pipeline name/i })).toHaveValue('');
    });
    expect(screen.getByTestId('upload-pipeline-dropzone')).toBeInTheDocument();
  });

  it('does not reset inputs after a failed upload', async () => {
    const { onClose } = renderDialog();
    const file = new File(['test'], 'my_pipeline.yaml', { type: 'application/yaml' });
    await dropFile(file);

    const nameInput = screen.getByRole('textbox', { name: /Pipeline name/i });
    await user.clear(nameInput);
    await user.type(nameInput, 'custom name');

    await user.click(screen.getByRole('button', { name: 'Upload' }));
    expect(onClose).toHaveBeenCalled();
    expect(screen.getByRole('textbox', { name: /Pipeline name/i })).toHaveValue('custom name');
  });
});
