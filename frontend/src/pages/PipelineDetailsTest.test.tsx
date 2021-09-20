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

import { render, screen } from '@testing-library/react';
import { graphlib } from 'dagre';
import * as JsYaml from 'js-yaml';
import React from 'react';
import { ApiExperiment } from 'src/apis/experiment';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { ApiRunDetail } from 'src/apis/run';
import { QUERY_PARAMS, RouteParams } from 'src/components/Router';
import * as v2PipelineSpec from 'src/data/test/mock_lightweight_python_functions_v2_pipeline.json';
import * as features from 'src/features';
import { Apis } from 'src/lib/Apis';
import TestUtils, { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import * as StaticGraphParser from '../lib/StaticGraphParser';
import { PageProps } from './Page';
import PipelineDetails from './PipelineDetails';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';

// This file is created in order to replace enzyme with react-testing-library gradually.
// The old test file is written using enzyme in PipelineDetails.test.tsx.
testBestPractices();
describe('switch between v1 and v2', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();

  let testPipeline: ApiPipeline = {};
  let testPipelineVersion: ApiPipelineVersion = {};
  let testRun: ApiRunDetail = {};

  function generateProps(fromRunSpec = false): PageProps {
    const match = {
      isExact: true,
      params: fromRunSpec
        ? {}
        : {
            [RouteParams.pipelineId]: testPipeline.id,
            [RouteParams.pipelineVersionId]:
              (testPipeline.default_version && testPipeline.default_version!.id) || '',
          },
      path: '',
      url: '',
    };
    const location = { search: fromRunSpec ? `?${QUERY_PARAMS.fromRunId}=test-run-id` : '' } as any;
    const pageProps = TestUtils.generatePageProps(
      PipelineDetails,
      location,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
    return pageProps;
  }
  const v1PipelineSpecTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: entry-point-test-
spec:
  arguments:
    parameters: []
  entrypoint: entry-point-test
  templates:
  - dag:
      tasks:
      - name: recurse-1
        template: recurse-1
      - name: leaf-1
        template: leaf-1
    name: start
  - dag:
      tasks:
      - name: start
        template: start
      - name: recurse-2
        template: recurse-2
    name: recurse-1
  - dag:
      tasks:
      - name: start
        template: start
      - name: leaf-2
        template: leaf-2
      - name: recurse-3
        template: recurse-3
    name: recurse-2
  - dag:
      tasks:
      - name: start
        template: start
      - name: recurse-1
        template: recurse-1
      - name: recurse-2
        template: recurse-2
    name: recurse-3
  - dag:
      tasks:
      - name: start
        template: start
    name: entry-point-test
  - container:
    name: leaf-1
  - container:
    name: leaf-2
    `;

  beforeAll(() => jest.spyOn(console, 'error').mockImplementation());

  beforeEach(() => {
    mockResizeObserver();

    testPipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test pipeline description',
      id: 'test-pipeline-id',
      name: 'test pipeline',
      parameters: [{ name: 'param1', value: 'value1' }],
      default_version: {
        id: 'test-pipeline-version-id',
        name: 'test-pipeline-version',
      },
    };

    testPipelineVersion = {
      id: 'test-pipeline-version-id',
      name: 'test-pipeline-version',
    };

    testRun = {
      run: {
        id: 'test-run-id',
        name: 'test run',
        pipeline_spec: {
          pipeline_id: 'run-pipeline-id',
        },
      },
    };

    jest.mock('src/lib/Apis', () => jest.fn());
    Apis.pipelineServiceApi.getPipeline = jest.fn().mockResolvedValue(testPipeline);
    Apis.pipelineServiceApi.getPipelineVersion = jest.fn().mockResolvedValue(testPipelineVersion);
    Apis.pipelineServiceApi.deletePipelineVersion = jest.fn();
    Apis.pipelineServiceApi.listPipelineVersions = jest
      .fn()
      .mockResolvedValue({ versions: [testPipelineVersion] });
    Apis.pipelineServiceApi.getTemplate = jest
      .fn()
      .mockResolvedValue({ template: 'test template' });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: 'test template' });

    Apis.runServiceApi.getRun = jest.fn().mockResolvedValue(testRun);
    Apis.experimentServiceApi.getExperiment = jest
      .fn()
      .mockResolvedValue({ id: 'test-experiment-id', name: 'test experiment' } as ApiExperiment);
  });

  afterEach(async () => {
    jest.resetAllMocks();
  });

  it('Show error if not valid v1 template and disabled v2 feature', async () => {
    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      return false;
    });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: 'bad graph' });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo:
          'Unable to convert string response from server to Argo workflow template: https://argoproj.github.io/argo-workflows/workflow-templates/',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show error if v1 template cannot generate graph and disabled v2 feature', async () => {
    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      return false;
    });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest.fn().mockResolvedValue({
      template: `    
      apiVersion: argoproj.io/v1alpha1
      kind: Workflow
      metadata:
        generateName: entry-point-test-
      `,
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad graph',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show error if not valid v2 template and enabled v2 feature', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2) {
        return true;
      }
      return false;
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'Unexpected token e in JSON at position 1',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show v1 page if valid v1 template and enabled v2 feature flag', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2) {
        return true;
      }
      return false;
    });

    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
    Apis.pipelineServiceApi.getTemplate = jest
      .fn()
      .mockResolvedValue({ template: v1PipelineSpecTemplate });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: v1PipelineSpecTemplate });

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });

  it('Show v1 page if valid v1 template and disabled v2 feature flag', async () => {
    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      return false;
    });
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: v1PipelineSpecTemplate });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    createGraphSpy.mockImplementation(() => new graphlib.Graph());

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });

  it('Show v2 page if valid v2 template and enabled v2 feature', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2) {
        return true;
      }
      return false;
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');
    Apis.pipelineServiceApi.getPipelineVersionTemplate = jest
      .fn()
      .mockResolvedValue({ template: JSON.stringify(v2PipelineSpec) });

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v2');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });
});
