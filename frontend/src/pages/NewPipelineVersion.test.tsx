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
import { ImportMethod, NewPipelineVersion } from './NewPipelineVersion';
import TestUtils from 'src/TestUtils';
import { shallow, ShallowWrapper, ReactWrapper } from 'enzyme';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';

class TestNewPipelineVersion extends NewPipelineVersion {
  public _pipelineSelectorClosed = super._pipelineSelectorClosed;
  public _onDropForTest = super._onDropForTest;
}

describe('NewPipelineVersion', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
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

  function generateProps(search?: string): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_PIPELINE_VERSION,
        search: search,
      } as any,
      match: '' as any,
      toolbarProps: {} as any,
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
    getPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApiV2, 'getPipeline')
      .mockImplementation(() => MOCK_PIPELINE);
    createPipelineVersionSpy = jest
      .spyOn(Apis.pipelineServiceApiV2, 'createPipelineVersion')
      .mockImplementation(() => MOCK_PIPELINE_VERSION);
    createPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApiV2, 'createPipeline')
      .mockImplementation(() => MOCK_PIPELINE);
    uploadPipelineSpy = jest
      .spyOn(Apis, 'uploadPipelineV2')
      .mockImplementation(() => MOCK_PIPELINE);
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree) {
      await tree.unmount();
    }
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  // New pipeline version page has two functionalities: creating a pipeline and creating a version under an existing pipeline.
  // Our tests will be divided into 3 parts: switching between creating pipeline or creating version; test pipeline creation; test pipeline version creation.

  describe('switching between creating pipeline and creating pipeline version', () => {
    it('creates pipeline is default when landing from pipeline list page', () => {
      tree = shallow(<TestNewPipelineVersion {...generateProps()} />);

      // When landing from pipeline list page, the default is to create pipeline
      expect(tree.state('newPipeline')).toBe(true);

      // Switch to create pipeline version
      tree.find('#createPipelineVersionUnderExistingPipelineBtn').simulate('change');
      expect(tree.state('newPipeline')).toBe(false);

      // Switch back
      tree.find('#createNewPipelineBtn').simulate('change');
      expect(tree.state('newPipeline')).toBe(true);
    });

    it('creates pipeline version is default when landing from pipeline details page', () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );

      // When landing from pipeline list page, the default is to create pipeline
      expect(tree.state('newPipeline')).toBe(false);

      // Switch to create pipeline version
      tree.find('#createNewPipelineBtn').simulate('change');
      expect(tree.state('newPipeline')).toBe(true);

      // Switch back
      tree.find('#createPipelineVersionUnderExistingPipelineBtn').simulate('change');
      expect(tree.state('newPipeline')).toBe(false);
    });
  });

  describe('creating version under an existing pipeline', () => {
    it('does not include any action buttons in the toolbar', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      await TestUtils.flushPromises();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [{ displayName: 'Pipeline Versions', href: '/pipeline_versions/new' }],
        pageTitle: 'New Pipeline',
      });
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating pipeline version name', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineVersionName')({
        target: { value: 'version name' },
      });

      expect(tree.state()).toHaveProperty('pipelineVersionName', 'version name');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating pipeline version description', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineVersionDescription')({
        target: { value: 'some description' },
      });

      expect(tree.state()).toHaveProperty('pipelineVersionDescription', 'some description');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating package url', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as TestNewPipelineVersion).handleChange('packageUrl')({
        target: { value: 'https://dummy' },
      });

      expect(tree.state()).toHaveProperty('packageUrl', 'https://dummy');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating code source', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as TestNewPipelineVersion).handleChange('codeSourceUrl')({
        target: { value: 'https://dummy' },
      });

      expect(tree.state()).toHaveProperty('codeSourceUrl', 'https://dummy');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it("sends a request to create a version when 'Create' is clicked", async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineVersionName')({
        target: { value: 'test version name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineVersionDescription')({
        target: { value: 'some description' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });

      await TestUtils.flushPromises();

      tree.find('#createNewPipelineOrVersionBtn').simulate('click');

      // The APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineVersionSpy).toHaveBeenLastCalledWith('original-run-pipeline-id', {
        pipeline_id: 'original-run-pipeline-id',
        display_name: 'test version name',
        description: 'some description',
        package_url: {
          pipeline_url: 'https://dummy_package_url',
        },
      });
    });

    // TODO(jingzhang36): test error dialog if creating pipeline version fails
  });

  describe('creating new pipeline', () => {
    it('renders the new pipeline page', async () => {
      tree = shallow(<TestNewPipelineVersion {...generateProps()} />);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();
    });

    it('switches between import methods', () => {
      tree = shallow(<TestNewPipelineVersion {...generateProps()} />);

      // Import method is URL by default
      expect(tree.state('importMethod')).toBe(ImportMethod.URL);

      // Click to import by local
      tree.find('#localPackageBtn').simulate('change');
      expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);

      // Click back to URL
      tree.find('#remotePackageBtn').simulate('change');
      expect(tree.state('importMethod')).toBe(ImportMethod.URL);
    });

    it('creates pipeline from url in single user mode', async () => {
      tree = shallow(
        <TestNewPipelineVersion {...generateProps()} buildInfo={{ apiServerMultiUser: false }} />,
      );

      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();

      tree.find('#createNewPipelineOrVersionBtn').simulate('click');
      // The APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('isPrivate', false);
      expect(tree.state()).toHaveProperty('newPipeline', true);
      expect(tree.state()).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        display_name: 'test pipeline name',
      });
    });

    it('creates private pipeline from url in multi user mode', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps()}
          namespace='ns'
          buildInfo={{ apiServerMultiUser: true }}
        />,
      );

      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();
      tree.find('#createNewPipelineOrVersionBtn').simulate('click');
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('isPrivate', true);
      expect(tree.state()).toHaveProperty('newPipeline', true);
      expect(tree.state()).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        display_name: 'test pipeline name',
        namespace: 'ns',
      });
    });

    it('creates shared pipeline from url in multi user mode', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps()}
          namespace='ns'
          buildInfo={{ apiServerMultiUser: true }}
        />,
      );

      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      tree.setState({ isPrivate: false });
      await TestUtils.flushPromises();
      tree.find('#createNewPipelineOrVersionBtn').simulate('click');
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('isPrivate', false);
      expect(tree.state()).toHaveProperty('newPipeline', true);
      expect(tree.state()).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        display_name: 'test pipeline name',
      });
    });

    it('creates pipeline from local file in single user mode', async () => {
      tree = shallow(
        <TestNewPipelineVersion {...generateProps()} buildInfo={{ apiServerMultiUser: false }} />,
      );

      // Set local file, pipeline name, pipeline description and click create
      tree.find('#localPackageBtn').simulate('change');
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      (tree.instance() as TestNewPipelineVersion)._onDropForTest([file]);
      tree.find('#createNewPipelineOrVersionBtn').simulate('click');

      tree.update();
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('isPrivate', false);
      expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        'test pipeline description',
        file,
        undefined,
      );
    });

    it('creates private pipeline from local file in multi user mode', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps()}
          namespace='ns'
          buildInfo={{ apiServerMultiUser: true }}
        />,
      );

      // Set local file, pipeline name, pipeline description and click create
      tree.find('#localPackageBtn').simulate('change');
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      (tree.instance() as TestNewPipelineVersion)._onDropForTest([file]);

      tree.find('#createNewPipelineOrVersionBtn').simulate('click');

      tree.update();
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('isPrivate', true);
      expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        'test pipeline description',
        file,
        'ns',
      );
    });

    it('creates shared pipeline from local file in multi user mode', async () => {
      tree = shallow(
        <TestNewPipelineVersion
          {...generateProps()}
          namespace='ns'
          buildInfo={{ apiServerMultiUser: true }}
        />,
      );

      // Set local file, pipeline name, pipeline description and click create
      tree.find('#localPackageBtn').simulate('change');
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as TestNewPipelineVersion).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      (tree.instance() as TestNewPipelineVersion)._onDropForTest([file]);
      tree.setState({ isPrivate: false });
      tree.find('#createNewPipelineOrVersionBtn').simulate('click');

      tree.update();
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('isPrivate', false);
      expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);

      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        'test pipeline description',
        file,
        undefined,
      );
    });

    it('allows updating pipeline version name', async () => {
      render(
        <TestNewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.pipeline_id}`)}
        />,
      );
      const pipelineVersionNameInput = await screen.findByLabelText(/Pipeline Version name/);
      fireEvent.change(pipelineVersionNameInput, { target: { value: 'new-pipeline-name' } });
      expect(pipelineVersionNameInput.closest('input')?.value).toBe('new-pipeline-name');
    });
  });
});
