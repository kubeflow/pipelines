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
import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import UploadPipelineDialog, { ImportMethod } from './UploadPipelineDialog';
import TestUtils from '../TestUtils';

type UploadPipelineDialogProps = React.ComponentProps<typeof UploadPipelineDialog>;

type UploadPipelineDialogState = UploadPipelineDialog['state'];

type UploadPipelineDialogRender = ReturnType<typeof render>;

class UploadPipelineDialogWrapper {
  private _instance: UploadPipelineDialog;
  private _renderResult: UploadPipelineDialogRender;

  public constructor(instance: UploadPipelineDialog, renderResult: UploadPipelineDialogRender) {
    this._instance = instance;
    this._renderResult = renderResult;
  }

  public instance(): UploadPipelineDialog {
    return this._instance;
  }

  public state<K extends keyof UploadPipelineDialogState>(
    key?: K,
  ): UploadPipelineDialogState | UploadPipelineDialogState[K] {
    const state = this._instance.state;
    return key ? state[key] : state;
  }

  public unmount(): void {
    this._renderResult.unmount();
  }

  public renderResult(): UploadPipelineDialogRender {
    return this._renderResult;
  }
}

function renderUploadDialog(props: UploadPipelineDialogProps): UploadPipelineDialogWrapper {
  const ref = React.createRef<UploadPipelineDialog>();
  const renderResult = render(<UploadPipelineDialog ref={ref} {...props} />);
  if (!ref.current) {
    throw new Error('UploadPipelineDialog instance not available');
  }
  return new UploadPipelineDialogWrapper(ref.current, renderResult);
}

describe('UploadPipelineDialog', () => {
  it('renders closed', () => {
    const { asFragment } = render(
      <UploadPipelineDialog open={false} onClose={vi.fn().mockResolvedValue(false)} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders open', () => {
    const { asFragment } = render(
      <UploadPipelineDialog open={true} onClose={vi.fn().mockResolvedValue(false)} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders an active dropzone', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    act(() => {
      wrapper.instance().setState({ dropzoneActive: true });
    });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders with a selected file to upload', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    const file = new File(['test'], 'test_upload_file.txt');
    act(() => {
      (wrapper.instance() as any)._onDrop([file]);
    });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders alternate UI for uploading via URL', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    act(() => {
      wrapper.instance().setState({ importMethod: ImportMethod.URL });
    });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('calls close callback with null and empty string when canceled', async () => {
    const spy = vi.fn().mockResolvedValue(false);
    render(<UploadPipelineDialog open={true} onClose={spy} />);
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenCalledWith(false, '', null, '', ImportMethod.LOCAL, true, '');
  });

  it('calls close callback with null and empty string when dialog is closed', async () => {
    const spy = vi.fn().mockResolvedValue(false);
    const wrapper = renderUploadDialog({ open: true, onClose: spy });
    act(() => {
      (wrapper.instance() as any)._uploadDialogClosed(false);
    });
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenCalledWith(false, '', null, '', ImportMethod.LOCAL, true, '');
    wrapper.unmount();
  });

  it('calls close callback with file name, file object, and description when confirmed', async () => {
    const spy = vi.fn().mockResolvedValue(false);
    const wrapper = renderUploadDialog({ open: true, onClose: spy });
    (wrapper.instance() as any)._dropzoneRef = { current: { open: () => null } };
    const file = new File(['test'], 'test file.txt');
    act(() => {
      (wrapper.instance() as any)._onDrop([file]);
    });
    act(() => {
      wrapper.instance().handleChange('uploadPipelineName')({
        target: { value: 'test name' },
      });
    });
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenLastCalledWith(
      true,
      'test name',
      expect.any(File),
      '',
      ImportMethod.LOCAL,
      true,
      '',
    );
    wrapper.unmount();
  });

  it('calls close callback with trimmed file url and pipeline name when confirmed', async () => {
    const spy = vi.fn().mockResolvedValue(false);
    const wrapper = renderUploadDialog({ open: true, onClose: spy });
    fireEvent.click(screen.getByLabelText('Import by URL'));
    act(() => {
      wrapper.instance().handleChange('fileUrl')({
        target: { value: '\n https://www.google.com/test-file.txt ' },
      });
      wrapper.instance().handleChange('uploadPipelineName')({
        target: { value: 'test name' },
      });
    });
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    await TestUtils.flushPromises();
    expect(spy).toHaveBeenLastCalledWith(
      true,
      'test name',
      null,
      'https://www.google.com/test-file.txt',
      ImportMethod.URL,
      true,
      '',
    );
    wrapper.unmount();
  });

  it('trims file extension for pipeline name suggestion', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    const file = new File(['test'], 'test_upload_file.tar.gz');
    act(() => {
      (wrapper.instance() as any)._onDrop([file]);
    });
    expect(wrapper.state('dropzoneActive')).toBe(false);
    expect(wrapper.state('uploadPipelineName')).toBe('test_upload_file');
    wrapper.unmount();
  });

  it('sets the import method based on which radio button is toggled', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    expect(wrapper.state('importMethod')).toBe(ImportMethod.LOCAL);

    fireEvent.click(screen.getByLabelText('Import by URL'));
    expect(wrapper.state('importMethod')).toBe(ImportMethod.URL);

    fireEvent.click(screen.getByLabelText('Upload a file'));
    expect(wrapper.state('importMethod')).toBe(ImportMethod.LOCAL);
    wrapper.unmount();
  });

  it('resets all state if the dialog is closed and the callback returns true', async () => {
    const spy = vi.fn().mockResolvedValue(true);
    const wrapper = renderUploadDialog({ open: true, onClose: spy });
    act(() => {
      wrapper.instance().setState({
        dropzoneActive: true,
        file: {} as File,
        fileName: 'test file name',
        fileUrl: 'https://some.url.com',
        importMethod: ImportMethod.URL,
        uploadPipelineDescription: 'test description',
        uploadPipelineName: 'test pipeline name',
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    await TestUtils.flushPromises();

    expect(wrapper.state('busy')).toBe(false);
    expect(wrapper.state('dropzoneActive')).toBe(false);
    expect(wrapper.state('file')).toBeNull();
    expect(wrapper.state('fileName')).toBe('');
    expect(wrapper.state('fileUrl')).toBe('');
    expect(wrapper.state('importMethod')).toBe(ImportMethod.LOCAL);
    expect(wrapper.state('uploadPipelineDescription')).toBe('');
    expect(wrapper.state('uploadPipelineName')).toBe('');
    wrapper.unmount();
  });

  it('does not reset the state if the dialog is closed and the callback returns false', async () => {
    const spy = vi.fn().mockResolvedValue(false);
    const wrapper = renderUploadDialog({ open: true, onClose: spy });
    act(() => {
      wrapper.instance().setState({
        dropzoneActive: true,
        file: {} as File,
        fileName: 'test file name',
        fileUrl: 'https://some.url.com',
        importMethod: ImportMethod.URL,
        uploadPipelineDescription: 'test description',
        uploadPipelineName: 'test pipeline name',
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    await TestUtils.flushPromises();

    expect(wrapper.state('dropzoneActive')).toBe(true);
    expect(wrapper.state('file')).toEqual({});
    expect(wrapper.state('fileName')).toBe('test file name');
    expect(wrapper.state('fileUrl')).toBe('https://some.url.com');
    expect(wrapper.state('importMethod')).toBe(ImportMethod.URL);
    expect(wrapper.state('uploadPipelineDescription')).toBe('test description');
    expect(wrapper.state('uploadPipelineName')).toBe('test pipeline name');
    expect(wrapper.state('busy')).toBe(false);
    wrapper.unmount();
  });

  it('sets an active dropzone on drag', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    act(() => {
      (wrapper.instance() as any)._onDropzoneDragEnter();
    });
    expect(wrapper.state('dropzoneActive')).toBe(true);
    wrapper.unmount();
  });

  it('sets an inactive dropzone on drag leave', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    act(() => {
      (wrapper.instance() as any)._onDropzoneDragLeave();
    });
    expect(wrapper.state('dropzoneActive')).toBe(false);
    wrapper.unmount();
  });

  it('sets a file object on drop', () => {
    const wrapper = renderUploadDialog({
      open: true,
      onClose: vi.fn().mockResolvedValue(false),
    });
    const file = new File(['test'], 'test upload file');
    act(() => {
      (wrapper.instance() as any)._onDrop([file]);
    });
    expect(wrapper.state('dropzoneActive')).toBe(false);
    expect(wrapper.state('uploadPipelineName')).toBe(file.name);
    wrapper.unmount();
  });
});
