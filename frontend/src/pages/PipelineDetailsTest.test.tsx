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

import { render, screen, waitFor } from '@testing-library/react';
import { graphlib } from 'dagre';
import * as JsYaml from 'js-yaml';
import React from 'react';
import { ApiExperiment } from 'src/apis/experiment';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { ApiRunDetail } from 'src/apis/run';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { QUERY_PARAMS, RouteParams } from 'src/components/Router';
import * as features from 'src/features';
import { Apis } from 'src/lib/Apis';
import TestUtils, { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import * as StaticGraphParser from 'src/lib/StaticGraphParser';
import { PageProps } from './Page';
import PipelineDetails from './PipelineDetails';
import fs from 'fs';

const V2_PIPELINESPEC_PATH = 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml';
const v2YamlTemplateString = fs.readFileSync(V2_PIPELINESPEC_PATH, 'utf8');

// This file is created in order to replace enzyme with react-testing-library gradually.
// The old test file is written using enzyme in PipelineDetails.test.tsx.
testBestPractices();
describe('switch between v1 and v2', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();

  let testV1Pipeline: ApiPipeline = {};
  let testV1PipelineVersion: ApiPipelineVersion = {};
  let testV1Run: ApiRunDetail = {};
  let testV2Pipeline: V2beta1Pipeline = {};
  let testV2PipelineVersion: V2beta1PipelineVersion = {};
  let testV2Run: V2beta1Run = {};

  function generateProps(fromRunSpec = false): PageProps {
    const match = {
      isExact: true,
      params: fromRunSpec
        ? {}
        : {
            [RouteParams.pipelineId]: testV1Pipeline.id,
            [RouteParams.pipelineVersionId]:
              (testV1Pipeline.default_version && testV1Pipeline.default_version!.id) || '',
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

    testV1Pipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test pipeline description',
      id: 'test-v1-pipeline-id',
      name: 'test v1 pipeline',
      parameters: [{ name: 'param1', value: 'value1' }],
      default_version: {
        id: 'test-v1-pipeline-version-id',
        name: 'test-v1-pipeline-version',
      },
    };

    testV1PipelineVersion = {
      id: 'test-v1-pipeline-version-id',
      name: 'test-v1-pipeline-version',
    };

    testV1Run = {
      run: {
        id: 'test-v1-run-id',
        name: 'test v1 run',
        pipeline_spec: {
          pipeline_id: 'run-v1-pipeline-id',
        },
      },
    };

    testV2Pipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test v2 pipeline description',
      pipeline_id: 'test-v2-pipeline-id',
      display_name: 'test v2 pipeline',
    };

    testV2PipelineVersion = {
      pipeline_id: 'test-v2-pipeline-id',
      pipeline_version_id: 'test-v2-pipeline-version-id',
      name: 'test-v2-pipeline-version',
      pipeline_spec: JsYaml.safeLoad(v2YamlTemplateString),
    };

    testV2Run = {
      run_id: 'test-v2-run-id',
      display_name: 'test v2 run',
      pipeline_version_reference: {},
    };

    jest.mock('src/lib/Apis', () => jest.fn());
    Apis.pipelineServiceApi.getPipeline = jest.fn().mockResolvedValue(testV1Pipeline);
    Apis.pipelineServiceApi.getPipelineVersion = jest.fn().mockResolvedValue(testV1PipelineVersion);
    Apis.pipelineServiceApi.deletePipelineVersion = jest.fn();
    Apis.pipelineServiceApi.listPipelineVersions = jest
      .fn()
      .mockResolvedValue({ versions: [testV1PipelineVersion] });
    Apis.runServiceApi.getRun = jest.fn().mockResolvedValue(testV1Run);

    Apis.pipelineServiceApiV2.getPipeline = jest.fn().mockResolvedValue(testV2Pipeline);
    Apis.pipelineServiceApiV2.getPipelineVersion = jest
      .fn()
      .mockResolvedValue(testV2PipelineVersion);
    Apis.pipelineServiceApiV2.listPipelineVersions = jest
      .fn()
      .mockResolvedValue({ pipeline_versions: [testV2PipelineVersion] });
    Apis.runServiceApiV2.getRun = jest.fn().mockResolvedValue(testV2Run);

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
    Apis.pipelineServiceApiV2.getPipelineVersion = jest.fn().mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: {
        apiVersion: 'bad apiversion',
        kind: 'bad kind',
      },
    });
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
    Apis.pipelineServiceApiV2.getPipelineVersion = jest.fn().mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
      },
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');
    render(<PipelineDetails {...generateProps()} />);

    await waitFor(() => {
      expect(createGraphSpy).toHaveBeenCalled();
    });

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
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    Apis.pipelineServiceApiV2.getPipelineVersion = jest.fn().mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: JsYaml.safeLoad(
        'spec:\n  arguments:\n    parameters:\n      - name: output\n',
      ),
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    TestUtils.makeErrorResponse(createGraphSpy, 'bad graph');
    render(<PipelineDetails {...generateProps()} />);

    await waitFor(() => {
      expect(createGraphSpy).toHaveBeenCalledTimes(0);
    });

    screen.getByTestId('pipeline-detail-v1');
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'Important infomation is missing. Pipeline Spec is invalid.',
        message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show v1 page if valid v1 template and enabled v2 feature flag', async () => {
    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
    Apis.pipelineServiceApiV2.getPipelineVersion = jest.fn().mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
      },
    });

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
    Apis.pipelineServiceApiV2.getPipelineVersion = jest.fn().mockResolvedValue({
      display_name: 'test-pipeline-version',
      pipeline_id: 'test-pipeline-id',
      pipeline_version_id: 'test-pipeline-version-id',
      pipeline_spec: {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
      },
    });
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
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });
    const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');
    createGraphSpy.mockImplementation(() => new graphlib.Graph());

    render(<PipelineDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    screen.getByTestId('pipeline-detail-v2');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
  });
});
