import React from 'react';
import { GettingStarted } from './GettingStarted';
import TestUtils, { diffHTML } from '../TestUtils';
import { render } from '@testing-library/react';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { ApiListPipelinesResponse } from '../apis/pipeline/api';

const PATH_BACKEND_CONFIG = '../../../backend/src/apiserver/config/sample_config.json';
const PATH_FRONTEND_CONFIG = '../config/sample_config_from_backend.json';
describe(`${PATH_FRONTEND_CONFIG}`, () => {
  it(`should be in sync with ${PATH_BACKEND_CONFIG}, if not please run "npm run sync-backend-sample-config" to update.`, () => {
    const configBackend = require(PATH_BACKEND_CONFIG);
    const configFrontend = require(PATH_FRONTEND_CONFIG);
    expect(configFrontend).toEqual(configBackend.map((sample: any) => sample.name));
  });
});

describe('GettingStarted page', () => {
  const updateBannerSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const pipelineListSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      GettingStarted,
      {} as any,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      null,
      updateToolbarSpy,
      null,
    );
  }

  beforeEach(() => {
    jest.resetAllMocks();
    const empty: ApiListPipelinesResponse = {
      pipelines: [],
      total_size: 0,
    };
    pipelineListSpy.mockImplementation(() => Promise.resolve(empty));
  });

  it('initially renders documentation', () => {
    const { container } = render(<GettingStarted {...generateProps()} />);
    expect(container).toMatchSnapshot();
  });

  it('renders documentation with pipeline deep link after querying demo pipelines', async () => {
    let count = 0;
    pipelineListSpy.mockImplementation(() => {
      ++count;
      const response: ApiListPipelinesResponse = {
        pipelines: [{ id: `pipeline-id-${count}` }],
      };
      return Promise.resolve(response);
    });
    const { container } = render(<GettingStarted {...generateProps()} />);
    const base = container.innerHTML;
    await TestUtils.flushPromises();
    expect(pipelineListSpy.mock.calls).toMatchSnapshot();
    expect(diffHTML({ base, update: container.innerHTML })).toMatchInlineSnapshot(`
      Snapshot Diff:
      - Expected
      + Received

      @@ --- --- @@
            <h2 id="demonstrations-and-tutorials">Demonstrations and Tutorials</h2>
            <p>This section contains demo and tutorial pipelines.</p>
            <p><strong>Demos</strong> - Try an end-to-end demonstration pipeline.</p>
            <ul>
              <li>
      -         <a href="#/pipelines" class="link">TFX pipeline demo with Keras</a>
      +         <a href="#/pipelines/details/pipeline-id-3?" class="link"
      +           >TFX pipeline demo with Keras</a
      +         >
                <ul>
                  <li>
                    Classification pipeline based on Keras.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/core/iris"
      @@ --- --- @@
                    >
                  </li>
                </ul>
              </li>
              <li>
      -         <a href="#/pipelines" class="link">TFX pipeline demo with Estimator</a>
      +         <a href="#/pipelines/details/pipeline-id-2?" class="link"
      +           >TFX pipeline demo with Estimator</a
      +         >
                <ul>
                  <li>
                    Classification pipeline with model analysis, based on a public
                    BigQuery dataset of taxicab trips.
                    <a
      @@ --- --- @@
                    >
                  </li>
                </ul>
              </li>
              <li>
      -         <a href="#/pipelines" class="link">XGBoost Pipeline demo</a>
      +         <a href="#/pipelines/details/pipeline-id-1?" class="link"
      +           >XGBoost Pipeline demo</a
      +         >
                <ul>
                  <li>
                    An example of end-to-end distributed training for an XGBoost model.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/core/xgboost_training_cm"
      @@ --- --- @@
              <strong>Tutorials</strong> - Learn pipeline concepts by following a
              tutorial.
            </p>
            <ul>
              <li>
      -         <a href="#/pipelines" class="link">Data passing in python components</a>
      +         <a href="#/pipelines/details/pipeline-id-4?" class="link"
      +           >Data passing in python components</a
      +         >
                <ul>
                  <li>
                    Shows how to pass data between python components.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/Data%20passing%20in%20python%20components"
    `);
  });

  it('fallbacks to show pipeline list page if request failed', async () => {
    let count = 0;
    pipelineListSpy.mockImplementation(
      (): Promise<ApiListPipelinesResponse> => {
        ++count;
        if (count === 1) {
          return Promise.reject(new Error('Mocked error'));
        } else if (count === 2) {
          // incomplete data
          return Promise.resolve({});
        } else if (count === 3) {
          // empty data
          return Promise.resolve({ pipelines: [], total_size: 0 });
        }
        return Promise.resolve({
          pipelines: [{ id: `pipeline-id-${count}` }],
          total_size: 1,
        });
      },
    );
    const { container } = render(<GettingStarted {...generateProps()} />);
    const base = container.innerHTML;
    await TestUtils.flushPromises();
    expect(diffHTML({ base, update: container.innerHTML })).toMatchInlineSnapshot(`
      Snapshot Diff:
      - Expected
      + Received

      @@ --- --- @@
              <strong>Tutorials</strong> - Learn pipeline concepts by following a
              tutorial.
            </p>
            <ul>
              <li>
      -         <a href="#/pipelines" class="link">Data passing in python components</a>
      +         <a href="#/pipelines/details/pipeline-id-4?" class="link"
      +           >Data passing in python components</a
      +         >
                <ul>
                  <li>
                    Shows how to pass data between python components.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/Data%20passing%20in%20python%20components"
    `);
  });
});
