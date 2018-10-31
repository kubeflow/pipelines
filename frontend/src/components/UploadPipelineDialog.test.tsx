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
import { shallow } from 'enzyme';
import UploadPipelineDialog from './UploadPipelineDialog';

describe('UploadPipelineDialog', () => {
  it('renders closed', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders open', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders an active dropzone', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.setState({ dropzoneActive: true });
    expect(tree).toMatchSnapshot();
  });

  it('renders with a selected file to upload', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.setState({ fileToUpload: true });
    expect(tree).toMatchSnapshot();
  });

  it('calls close callback with null and empty string when canceled', () => {
    const spy = jest.fn();
    const tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    tree.find('#cancelUploadBtn').at(0).simulate('click');
    expect(spy).toHaveBeenCalledWith('', null, '');
  });

  it('calls close callback with null and empty string when dialog is closed', () => {
    const spy = jest.fn();
    const tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    tree.find('WithStyles(Dialog)').at(0).simulate('close');
    expect(spy).toHaveBeenCalledWith('', null, '');
  });

  it('calls close callback with file name, file object, and descriptio when confirmed', () => {
    const spy = jest.fn();
    const tree = shallow(<UploadPipelineDialog open={false} onClose={spy} />);
    (tree.instance() as any)._dropzoneRef = { current: { open: () => null } };
    (tree.instance() as UploadPipelineDialog).handleChange('uploadPipelineName')({ target: { value: 'test name' } });
    tree.find('#confirmUploadBtn').at(0).simulate('click');
    expect(spy).toHaveBeenLastCalledWith('test name', null, '');
  });

  it('sets an active dropzone on drag', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.find('#dropZone').simulate('dragEnter');
    expect(tree.state()).toHaveProperty('dropzoneActive', true);
  });

  it('sets an inactive dropzone on drag leave', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    tree.find('#dropZone').simulate('dragLeave');
    expect(tree.state()).toHaveProperty('dropzoneActive', false);
  });

  it('sets a file object on drop', () => {
    const tree = shallow(<UploadPipelineDialog open={false} onClose={jest.fn()} />);
    const file = { name: 'test upload file' };
    tree.find('#dropZone').simulate('drop', [file]);
    expect(tree.state()).toHaveProperty('dropzoneActive', false);
    expect(tree.state()).toHaveProperty('uploadPipelineName', file.name);
  });
});
