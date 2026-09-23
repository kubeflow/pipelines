/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 */
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import * as JsYaml from 'js-yaml';
import { MemoryRouter } from 'react-router';
import { RouteParams } from 'src/components/Router';
import { mockResizeObserver } from 'src/TestUtils';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import RunDetailsRouter from './RunDetailsRouter';

it('renders a deleted-version run using the saved spec requested through the generated client', async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const requests: URL[] = [];
  const run = {
    run_id: 'saved-run',
    display_name: 'Saved run',
    state: 'SUCCEEDED',
    pipeline_version_reference: { pipeline_id: 'pipeline', pipeline_version_id: 'deleted-version' },
  };
  mockResizeObserver();
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      const url = new URL(input, 'http://localhost');
      requests.push(url);
      if (url.pathname.endsWith('/pipelines/pipeline/versions/deleted-version')) {
        return Response.json({ message: 'Pipeline version not found' }, { status: 404 });
      }
      if (url.pathname.endsWith('/runs/saved-run/tasks')) {
        return Response.json({ tasks: [] });
      }
      if (url.pathname.endsWith('/runs/saved-run')) {
        return Response.json(
          url.searchParams.get('view') === 'FULL'
            ? { ...run, pipeline_spec: JsYaml.load(v2YamlTemplateString) }
            : run,
        );
      }
      throw new Error(`Unexpected request: ${url}`);
    }),
  );
  const updateBanner = vi.fn();
  const view = render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <RunDetailsRouter
          navigate={vi.fn()}
          location={{
            pathname: '/runs/details/saved-run',
            search: '',
            hash: '',
            state: null,
            key: 'test',
          }}
          params={{ [RouteParams.runId]: 'saved-run' }}
          toolbarProps={{ actions: {}, breadcrumbs: [], pageTitle: '' }}
          updateBanner={updateBanner}
          updateDialog={vi.fn()}
          updateSnackbar={vi.fn()}
          updateToolbar={vi.fn()}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  try {
    expect(await screen.findByRole('button', { name: 'Pipeline Spec' })).toBeInTheDocument();
    const fullRequests = requests.filter((url) => url.searchParams.get('view') === 'FULL');
    expect(fullRequests).toHaveLength(1);
    expect(fullRequests[0].searchParams.has('experiment_id')).toBe(false);
    expect(
      screen.queryByText('Unable to load run details. Refresh this page to retry.'),
    ).toBeNull();
    expect(updateBanner).not.toHaveBeenCalledWith(expect.objectContaining({ mode: 'error' }));
  } finally {
    view.unmount();
    client.clear();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  }
});
