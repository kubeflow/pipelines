/*
 * Copyright 2019 Google LLC
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
import NewPipelineVersion, { ImportMethod } from './NewPipelineVersion';
import TestUtils, { defaultToolbarProps } from '../TestUtils';
import { shallow, ShallowWrapper, ReactWrapper } from 'enzyme';
import { Apis } from '../lib/Apis';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import { ApiResourceType } from '../apis/pipeline';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (component: React.ComponentClass) => {
    component.defaultProps = { ...component.defaultProps, t: (key: string) => key };
    return component;
  },
}));

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
    id: 'original-run-pipeline-id',
    name: 'original mock pipeline name',
    default_version: {
      id: 'original-run-pipeline-version-id',
      name: 'original mock pipeline version name',
      resource_references: [
        {
          key: {
            id: 'original-run-pipeline-id',
            type: ApiResourceType.PIPELINE,
          },
          relationship: 1,
        },
      ],
    },
  };

  let MOCK_PIPELINE_VERSION = {
    id: 'original-run-pipeline-version-id',
    name: 'original mock pipeline version name',
    resource_references: [
      {
        key: {
          id: 'original-run-pipeline-id',
          type: ApiResourceType.PIPELINE,
        },
        relationship: 1,
      },
    ],
  };

  function generateProps(search?: string): any {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_PIPELINE_VERSION,
        search: search ?? '',
      },
      toolbarProps: defaultToolbarProps(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
    getPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApi, 'getPipeline')
      .mockImplementation(() => MOCK_PIPELINE);
    createPipelineVersionSpy = jest
      .spyOn(Apis.pipelineServiceApi, 'createPipelineVersion')
      .mockImplementation(() => MOCK_PIPELINE_VERSION);
    createPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApi, 'createPipeline')
      .mockImplementation(() => MOCK_PIPELINE);
    uploadPipelineSpy = jest.spyOn(Apis, 'uploadPipeline').mockImplementation(() => MOCK_PIPELINE);
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
      tree = shallow(<NewPipelineVersion {...generateProps()} />);

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
        <NewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`)}
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
        <NewPipelineVersion
          t={(key: any) => key}
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`)}
        />,
      );
      await TestUtils.flushPromises();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [
          { displayName: 'pipelines:pipelineVersions', href: '/pipeline_versions/new' },
        ],
        pageTitle: 'pipelines:uploadPipelineTitle',
        t: expect.any(Function),
      });
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating pipeline version name', async () => {
      tree = shallow(
        <NewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as any).handleChange('pipelineVersionName')({
        target: { value: 'version name' },
      });

      expect(tree.state()).toHaveProperty('pipelineVersionName', 'version name');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating package url', async () => {
      tree = shallow(
        <NewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as any).handleChange('packageUrl')({
        target: { value: 'https://dummy' },
      });

      expect(tree.state()).toHaveProperty('packageUrl', 'https://dummy');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it('allows updating code source', async () => {
      tree = shallow(
        <NewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as any).handleChange('codeSourceUrl')({
        target: { value: 'https://dummy' },
      });

      expect(tree.state()).toHaveProperty('codeSourceUrl', 'https://dummy');
      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
    });

    it("sends a request to create a version when 'Create' is clicked", async () => {
      tree = shallow(
        <NewPipelineVersion
          {...generateProps(`?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`)}
        />,
      );
      await TestUtils.flushPromises();

      (tree.instance() as any).handleChange('pipelineVersionName')({
        target: { value: 'test version name' },
      });
      (tree.instance() as any).handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();

      tree.find('#createNewPipelineOrVersionBtn').simulate('click');
      // The APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(createPipelineVersionSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineVersionSpy).toHaveBeenLastCalledWith({
        code_source_url: '',
        name: 'test version name',
        package_url: {
          pipeline_url: 'https://dummy_package_url',
        },
        resource_references: [
          {
            key: {
              id: MOCK_PIPELINE.id,
              type: ApiResourceType.PIPELINE,
            },
            relationship: 1,
          },
        ],
      });
    });

    // TODO(jingzhang36): test error dialog if creating pipeline version fails
  });

  describe('creating new pipeline', () => {
    it('renders the new pipeline page', async () => {
      tree = shallow(<NewPipelineVersion {...generateProps()} />);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();
    });

    it('switches between import methods', () => {
      tree = shallow(<NewPipelineVersion {...generateProps()} />);

      // Import method is URL by default
      expect(tree.state('importMethod')).toBe(ImportMethod.URL);

      // Click to import by local
      tree.find('#localPackageBtn').simulate('change');
      expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);

      // Click back to URL
      tree.find('#remotePackageBtn').simulate('change');
      expect(tree.state('importMethod')).toBe(ImportMethod.URL);
    });

    it('creates pipeline from url', async () => {
      tree = shallow(<NewPipelineVersion {...generateProps()} />);

      (tree.instance() as any).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as any).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      (tree.instance() as any).handleChange('packageUrl')({
        target: { value: 'https://dummy_package_url' },
      });
      await TestUtils.flushPromises();

      tree.find('#createNewPipelineOrVersionBtn').simulate('click');
      // The APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state()).toHaveProperty('newPipeline', true);
      expect(tree.state()).toHaveProperty('importMethod', ImportMethod.URL);
      expect(createPipelineSpy).toHaveBeenCalledTimes(1);
      expect(createPipelineSpy).toHaveBeenLastCalledWith({
        description: 'test pipeline description',
        name: 'test pipeline name',
        url: {
          pipeline_url: 'https://dummy_package_url',
        },
      });
    });

    it('creates pipeline from local file', async () => {
      tree = shallow(<NewPipelineVersion {...generateProps()} />);

      // Set local file, pipeline name, pipeline description and click create
      tree.find('#localPackageBtn').simulate('change');
      (tree.instance() as any).handleChange('pipelineName')({
        target: { value: 'test pipeline name' },
      });
      (tree.instance() as any).handleChange('pipelineDescription')({
        target: { value: 'test pipeline description' },
      });
      const file = new File(['file contents'], 'file_name', { type: 'text/plain' });
      (tree.instance() as any)._onDropForTest([file]);
      tree.find('#createNewPipelineOrVersionBtn').simulate('click');

      tree.update();
      await TestUtils.flushPromises();

      expect(tree.state('importMethod')).toBe(ImportMethod.LOCAL);
      expect(uploadPipelineSpy).toHaveBeenLastCalledWith(
        'test pipeline name',
        'test pipeline description',
        file,
      );
      expect(createPipelineSpy).not.toHaveBeenCalled();
    });
  });
});
