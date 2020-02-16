import React from 'react';
import { GettingStarted } from './GettingStarted';
import TestUtils, { formatHTML } from '../TestUtils';
import { render } from '@testing-library/react';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { ApiListPipelinesResponse } from 'src/apis/pipeline/api';
import snapshotDiff from 'snapshot-diff';

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
    const initialHtml = container.innerHTML;
    await TestUtils.flushPromises();
    expect(
      snapshotDiff(formatHTML(initialHtml), formatHTML(container.innerHTML)),
    ).toMatchSnapshot();
  });
});
