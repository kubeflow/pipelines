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
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div
          class="page kfp-start-page"
        >
          <div>
            <br />
            <h2
              id="build-your-own-pipeline-with"
            >
              Build your own pipeline with
            </h2>
            <ul>
              <li>
                TensorFlow Extended (TFX) 
                <a
                  class="link"
                  href="https://www.tensorflow.org/tfx/guide"
                  rel="noopener"
                  target="_blank"
                >
                  SDK
                </a>
                 with end-to-end ML Pipeline Template (
                <a
                  class="link"
                  href="https://console.cloud.google.com/mlengine/notebooks/deploy-notebook?q=download_url%3Dhttps%253A%252F%252Fraw.githubusercontent.com%252Ftensorflow%252Ftfx%252Fmaster%252Fdocs%252Ftutorials%252Ftfx%252Ftemplate.ipynb"
                  rel="noopener"
                  target="_blank"
                >
                  Open TF 2.1 Notebook
                </a>
                )
              </li>
              <li>
                Kubeflow Pipelines 
                <a
                  class="link"
                  href="https://www.kubeflow.org/docs/pipelines/sdk/"
                  rel="noopener"
                  target="_blank"
                >
                  SDK
                </a>
              </li>
            </ul>
            <br />
            <h2
              id="demonstrations-and-tutorials"
            >
              Demonstrations and Tutorials
            </h2>
            <p>
              This section contains demo and tutorial pipelines.
            </p>
            <p>
              <strong>
                Demos
              </strong>
               - Try an end-to-end demonstration pipeline.
            </p>
            <ul>
              <li>
                <a
                  class="link"
                  href="#/pipelines"
                >
                  TFX pipeline demo with Keras
                </a>
                 
                <ul>
                  <li>
                    Classification pipeline based on Keras. 
                    <a
                      class="link"
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/core/iris"
                      rel="noopener"
                      target="_blank"
                    >
                      source code
                    </a>
                  </li>
                </ul>
              </li>
              <li>
                <a
                  class="link"
                  href="#/pipelines"
                >
                  TFX pipeline demo with Estimator
                </a>
                 
                <ul>
                  <li>
                    Classification pipeline with model analysis, based on a public BigQuery dataset of taxicab trips. 
                    <a
                      class="link"
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/core/parameterized_tfx_oss"
                      rel="noopener"
                      target="_blank"
                    >
                      source code
                    </a>
                  </li>
                </ul>
              </li>
            </ul>
            <br />
            <p>
              <strong>
                Tutorials
              </strong>
               - Learn pipeline concepts by following a tutorial.
            </p>
            <ul>
              <li>
                <a
                  class="link"
                  href="#/pipelines"
                >
                  Data passing in python components
                </a>
                 
                <ul>
                  <li>
                    Shows how to pass data between python components. 
                    <a
                      class="link"
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/Data%20passing%20in%20python%20components"
                      rel="noopener"
                      target="_blank"
                    >
                      source code
                    </a>
                  </li>
                </ul>
              </li>
              <li>
                <a
                  class="link"
                  href="#/pipelines"
                >
                  DSL - Control structures
                </a>
                 
                <ul>
                  <li>
                    Shows how to use conditional execution and exit handlers. 
                    <a
                      class="link"
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/DSL%20-%20Control%20structures"
                      rel="noopener"
                      target="_blank"
                    >
                      source code
                    </a>
                  </li>
                </ul>
              </li>
            </ul>
            <p>
              Want to learn more? 
              <a
                class="link"
                href="https://www.kubeflow.org/docs/pipelines/tutorials/"
                rel="noopener"
                target="_blank"
              >
                Learn from sample and tutorial pipelines.
              </a>
            </p>
          </div>
        </div>
      </div>
    `);
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
    expect(pipelineListSpy.mock.calls).toMatchInlineSnapshot(`
      Array [
        Array [
          undefined,
          10,
          undefined,
          "%7B%22predicates%22%3A%5B%7B%22key%22%3A%22name%22%2C%22op%22%3A%22EQUALS%22%2C%22string_value%22%3A%22%5BDemo%5D%20TFX%20-%20Taxi%20tip%20prediction%20model%20trainer%22%7D%5D%7D",
        ],
        Array [
          undefined,
          10,
          undefined,
          "%7B%22predicates%22%3A%5B%7B%22key%22%3A%22name%22%2C%22op%22%3A%22EQUALS%22%2C%22string_value%22%3A%22%5BDemo%5D%20TFX%20-%20Iris%20classification%20pipeline%22%7D%5D%7D",
        ],
        Array [
          undefined,
          10,
          undefined,
          "%7B%22predicates%22%3A%5B%7B%22key%22%3A%22name%22%2C%22op%22%3A%22EQUALS%22%2C%22string_value%22%3A%22%5BTutorial%5D%20Data%20passing%20in%20python%20components%22%7D%5D%7D",
        ],
        Array [
          undefined,
          10,
          undefined,
          "%7B%22predicates%22%3A%5B%7B%22key%22%3A%22name%22%2C%22op%22%3A%22EQUALS%22%2C%22string_value%22%3A%22%5BTutorial%5D%20DSL%20-%20Control%20structures%22%7D%5D%7D",
        ],
      ]
    `);
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
      +         <a href="#/pipelines/details/pipeline-id-2?" class="link"
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
      +         <a href="#/pipelines/details/pipeline-id-1?" class="link"
      +           >TFX pipeline demo with Estimator</a
      +         >
                <ul>
                  <li>
                    Classification pipeline with model analysis, based on a public
                    BigQuery dataset of taxicab trips.
                    <a
      @@ --- --- @@
              <strong>Tutorials</strong> - Learn pipeline concepts by following a
              tutorial.
            </p>
            <ul>
              <li>
      -         <a href="#/pipelines" class="link">Data passing in python components</a>
      +         <a href="#/pipelines/details/pipeline-id-3?" class="link"
      +           >Data passing in python components</a
      +         >
                <ul>
                  <li>
                    Shows how to pass data between python components.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/Data%20passing%20in%20python%20components"
      @@ --- --- @@
                    >
                  </li>
                </ul>
              </li>
              <li>
      -         <a href="#/pipelines" class="link">DSL - Control structures</a>
      +         <a href="#/pipelines/details/pipeline-id-4?" class="link"
      +           >DSL - Control structures</a
      +         >
                <ul>
                  <li>
                    Shows how to use conditional execution and exit handlers.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/DSL%20-%20Control%20structures"
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
                    >
                  </li>
                </ul>
              </li>
              <li>
      -         <a href="#/pipelines" class="link">DSL - Control structures</a>
      +         <a href="#/pipelines/details/pipeline-id-4?" class="link"
      +           >DSL - Control structures</a
      +         >
                <ul>
                  <li>
                    Shows how to use conditional execution and exit handlers.
                    <a
                      href="https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/DSL%20-%20Control%20structures"
    `);
  });
});
