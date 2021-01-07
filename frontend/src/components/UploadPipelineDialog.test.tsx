/*
 * Copyright 2018 Google LLC
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
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import UploadPipelineDialog, { ImportMethod } from './UploadPipelineDialog';
import TestUtils from '../TestUtils';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate HoC receive the t function as a prop
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: () => '' };
    return Component;
  },
}));
describe('UploadPipelineDialog', () => {
  let tree: ReactWrapper | ShallowWrapper;

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
  });

  it('renders closed', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders open', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders an active dropzone', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.setState({ dropzoneActive: true });
    expect(tree).toMatchSnapshot();
  });

  it('renders with a selected file to upload', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.setState({ fileToUpload: true });
    expect(tree).toMatchSnapshot();
  });

  it('renders alternate UI for uploading via URL', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.setState({ importMethod: ImportMethod.URL });
    expect(tree).toMatchSnapshot();
  });

  it('calls close callback with null and empty string when canceled', () => {
    const spy = jest.fn();
    tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    tree.find('#cancelUploadBtn').simulate('click');
    expect(spy).toHaveBeenCalledWith(false, '', null, '', ImportMethod.LOCAL, '');
  });

  it('calls close callback with null and empty string when dialog is closed', () => {
    const spy = jest.fn();
    tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    tree.find('WithStyles(Dialog)').simulate('close');
    expect(spy).toHaveBeenCalledWith(false, '', null, '', ImportMethod.LOCAL, '');
  });

  it('calls close callback with file name, file object, and description when confirmed', () => {
    const spy = jest.fn();
    tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    (tree.instance() as any)._dropzoneRef = { current: { open: () => null } };
    (tree.instance() as UploadPipelineDialog).handleChange('uploadPipelineName')({
      target: { value: 'test name' },
    });
    tree.find('#confirmUploadBtn').simulate('click');
    expect(spy).toHaveBeenLastCalledWith(true, 'test name', null, '', ImportMethod.LOCAL, '');
  });

  it('calls close callback with trimmed file url and pipeline name when confirmed', () => {
    const spy = jest.fn();
    tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    // Click 'Import by URL'
    tree.find('#uploadFromUrlBtn').simulate('change');
    (tree.instance() as UploadPipelineDialog).handleChange('fileUrl')({
      target: { value: '\n https://www.google.com/test-file.txt ' },
    });
    (tree.instance() as UploadPipelineDialog).handleChange('uploadPipelineName')({
      target: { value: 'test name' },
    });
    tree.find('#confirmUploadBtn').simulate('click');
    expect(spy).toHaveBeenLastCalledWith(
      true,
      'test name',
      null,
      'https://www.google.com/test-file.txt',
      ImportMethod.URL,
      '',
    );
  });

  it('trims file extension for pipeline name suggestion', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    const file = { name: 'test_upload_file.tar.gz' };
    tree.find('#dropZone').simulate('drop', [file]);
    expect(tree.state()).toHaveProperty('dropzoneActive', false);
    expect(tree.state()).toHaveProperty('uploadPipelineName', 'test_upload_file');
  });

  it('sets the import method based on which radio button is toggled', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    // Import method is LOCAL by default
    expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);

    // Click 'Import by URL'
    tree.find('#uploadFromUrlBtn').simulate('change');
    expect(tree.state('importMethod')).toBe(ImportMethod.URL);

    // Click back to default, 'Upload a file'
    tree.find('#uploadLocalFileBtn').simulate('change');
    expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);
  });

  it('resets all state if the dialog is closed and the callback returns true', async () => {
    const spy = jest.fn(() => true);

    tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    tree.setState({
      busy: true,
      dropzoneActive: true,
      file: {},
      fileName: 'test file name',
      fileUrl: 'https://some.url.com',
      importMethod: ImportMethod.URL,
      uploadPipelineDescription: 'test description',
      uploadPipelineName: 'test pipeline name',
    });

    tree.find('#confirmUploadBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(tree.state('busy')).toBe(false);
    expect(tree.state('dropzoneActive')).toBe(false);
    expect(tree.state('file')).toBeNull();
    expect(tree.state('fileName')).toBe('');
    expect(tree.state('fileUrl')).toBe('');
    expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);
    expect(tree.state('uploadPipelineDescription')).toBe('');
    expect(tree.state('uploadPipelineName')).toBe('');
  });

  it('does not reset the state if the dialog is closed and the callback returns false', async () => {
    const spy = jest.fn(() => false);

    tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    tree.setState({
      busy: true,
      dropzoneActive: true,
      file: {},
      fileName: 'test file name',
      fileUrl: 'https://some.url.com',
      importMethod: ImportMethod.URL,
      uploadPipelineDescription: 'test description',
      uploadPipelineName: 'test pipeline name',
    });

    tree.find('#confirmUploadBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(tree.state('dropzoneActive')).toBe(true);
    expect(tree.state('file')).toEqual({});
    expect(tree.state('fileName')).toBe('test file name');
    expect(tree.state('fileUrl')).toBe('https://some.url.com');
    expect(tree.state('importMethod')).toBe(ImportMethod.URL);
    expect(tree.state('uploadPipelineDescription')).toBe('test description');
    expect(tree.state('uploadPipelineName')).toBe('test pipeline name');
    // 'busy' is set to false regardless upon the callback returning
    expect(tree.state('busy')).toBe(false);
  });

  it('sets an active dropzone on drag', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.find('#dropZone').simulate('dragEnter');
    expect(tree.state()).toHaveProperty('dropzoneActive', true);
  });

  it('sets an inactive dropzone on drag leave', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.find('#dropZone').simulate('dragLeave');
    expect(tree.state()).toHaveProperty('dropzoneActive', false);
  });

  it('sets a file object on drop', () => {
    tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    const file = { name: 'test upload file' };
    tree.find('#dropZone').simulate('drop', [file]);
    expect(tree.state()).toHaveProperty('dropzoneActive', false);
    expect(tree.state()).toHaveProperty('uploadPipelineName', file.name);
  });
});
