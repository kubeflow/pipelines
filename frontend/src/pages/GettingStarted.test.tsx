import React from 'react';
import GettingStarted from './GettingStarted';
import TestUtils, { diffHTML } from '../TestUtils';
import { render } from '@testing-library/react';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { ApiListPipelinesResponse } from '../apis/pipeline/api';

jest.mock('react-i18next', () => ({
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: ((key: string) => key) as any };
    return Component;
  },

  useTranslation: () => {
    return {
      t: (key: string) => key as any,
    };
  },
}));
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
      Compared values have no visual difference.
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
      Compared values have no visual difference.
    `);
  });
});
