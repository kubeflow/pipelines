/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 */
import { render, screen } from '@testing-library/react';
import { CommonTestWrapper } from 'src/TestWrapper';
import RunList from './RunList';

function renderWithVersionError(status: number) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      const url = new URL(input, 'http://localhost');
      if (url.pathname.endsWith('/runs')) {
        return Response.json({
          runs: [
            {
              run_id: 'saved-run',
              display_name: 'Saved run',
              state: 'SUCCEEDED',
              pipeline_version_reference: {
                pipeline_id: 'pipeline',
                pipeline_version_id: 'version',
              },
            },
          ],
        });
      }
      if (url.pathname.endsWith('/experiments')) {
        return Response.json({ experiments: [] });
      }
      if (url.pathname.endsWith('/pipelines/pipeline/versions/version')) {
        return Response.json({ message: 'Pipeline version unavailable' }, { status });
      }
      throw new Error(`Unexpected request: ${url}`);
    }),
  );
  return render(
    <CommonTestWrapper>
      <RunList
        navigate={vi.fn()}
        location={{ pathname: '/runs', search: '', hash: '', state: null, key: 'test' }}
        params={{}}
        onError={vi.fn()}
      />
    </CommonTestWrapper>,
  );
}

afterEach(() => vi.unstubAllGlobals());

it.each([403, 500])(
  'still reports HTTP %s errors when looking up a pipeline version',
  async (status) => {
    renderWithVersionError(status);
    expect(
      await screen.findByLabelText(/Failed to get associated pipeline version/),
    ).toBeInTheDocument();
    expect(screen.queryByText('Unavailable')).toBeNull();
  },
);

it('keeps a run browsable without an error warning when its pipeline version was deleted', async () => {
  renderWithVersionError(404);
  expect(await screen.findByText('Unavailable')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Saved run' })).toHaveAttribute(
    'href',
    '/runs/details/saved-run',
  );
  expect(screen.queryByLabelText(/Failed to get associated pipeline version/)).toBeNull();
});
