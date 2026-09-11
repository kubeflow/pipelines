const {
  canReuseExistingData,
  createRecurringRun,
  createRelatedTaskArtifact,
  createRun,
  listComparableRunIds,
  selectComparableRunIds,
  uploadPipeline,
} = require('./seed-data.js');

describe('UI smoke data seeding', () => {
  it('creates a pipeline with the raw Pipeline body expected by the gateway', async () => {
    const request = vi.fn().mockResolvedValue({ pipeline_id: 'pipeline-1' });

    await expect(uploadPipeline('Smoke Pipeline', 'Smoke description', request)).resolves.toEqual({
      pipeline_id: 'pipeline-1',
    });
    expect(request).toHaveBeenCalledWith('POST', '/apis/v2beta1/pipelines', {
      description: 'Smoke description',
      display_name: 'Smoke Pipeline',
    });
  });

  it('creates a run with an embedded spec and required runtime config', async () => {
    const request = vi.fn().mockResolvedValue({ run_id: 'run-1' });

    await expect(createRun('Smoke Run', 'pipeline-1', 'experiment-1', request)).resolves.toEqual({
      run_id: 'run-1',
    });
    expect(request).toHaveBeenCalledWith(
      'POST',
      '/apis/v2beta1/runs',
      expect.objectContaining({
        experiment_id: 'experiment-1',
        pipeline_spec: expect.objectContaining({
          pipelineInfo: expect.objectContaining({ name: 'smoke-run' }),
          schemaVersion: '2.1.0',
        }),
        runtime_config: { pipeline_root: 'minio://mlpipeline/ui-smoke/smoke-run' },
      }),
    );
    expect(request.mock.calls[0][2]).not.toHaveProperty('pipeline_version_reference');
    expect(
      selectComparableRunIds([
        { run_id: 'run-1', pipeline_spec: request.mock.calls[0][2].pipeline_spec },
      ]),
    ).toEqual(['run-1']);
  });

  it('creates a recurring run with an embedded spec and required runtime config', async () => {
    const request = vi.fn().mockResolvedValue({ recurring_run_id: 'recurring-1' });

    await createRecurringRun('Smoke Schedule', 'pipeline-1', 'experiment-1', request);

    expect(request).toHaveBeenCalledWith(
      'POST',
      '/apis/v2beta1/recurringruns',
      expect.objectContaining({
        pipeline_spec: expect.objectContaining({
          pipelineInfo: expect.objectContaining({ name: 'smoke-schedule' }),
          schemaVersion: '2.1.0',
        }),
        runtime_config: { pipeline_root: 'minio://mlpipeline/ui-smoke/smoke-schedule' },
      }),
    );
    expect(request.mock.calls[0][2]).not.toHaveProperty('pipeline_version_reference');
  });

  it('creates and verifies a native artifact/task relationship', async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce({ tasks: [] })
      .mockResolvedValueOnce({ task_id: 'task-1' })
      .mockResolvedValueOnce({ artifact_id: 'artifact-1' })
      .mockResolvedValueOnce({
        artifact_tasks: [{ artifact_id: 'artifact-1', task_id: 'task-1' }],
      });

    await expect(createRelatedTaskArtifact('run-1', request)).resolves.toEqual({
      artifactId: 'artifact-1',
      taskId: 'task-1',
    });
    expect(request).toHaveBeenNthCalledWith(
      1,
      'GET',
      '/apis/v2beta1/runs/run-1/tasks?page_size=200',
    );
    expect(request).toHaveBeenNthCalledWith(2, 'POST', '/apis/v2beta1/runs/run-1/tasks', {
      display_name: 'UI Smoke Related Task',
      name: 'ui-smoke-related-task',
      scope_path: 'root.ui-smoke-related-task',
      state: 'SUCCEEDED',
      type: 'RUNTIME',
    });
    expect(request).toHaveBeenNthCalledWith(
      3,
      'POST',
      '/apis/v2beta1/artifacts',
      expect.objectContaining({
        producer_key: 'ui_smoke_output',
        run_id: 'run-1',
        task_id: 'task-1',
      }),
    );
    expect(request).toHaveBeenNthCalledWith(
      4,
      'GET',
      '/apis/v2beta1/artifact_tasks?artifact_ids=artifact-1&page_size=1',
    );
  });

  it('reuses an existing native artifact/task relationship', async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce({ tasks: [{ name: 'ui-smoke-related-task', task_id: 'task-1' }] })
      .mockResolvedValueOnce({
        artifact_tasks: [{ artifact_id: 'artifact-1', task_id: 'task-1' }],
      });

    await expect(createRelatedTaskArtifact('run-1', request)).resolves.toEqual({
      artifactId: 'artifact-1',
      taskId: 'task-1',
    });
    expect(request).toHaveBeenCalledTimes(2);
    expect(request).toHaveBeenNthCalledWith(
      2,
      'GET',
      '/apis/v2beta1/artifact_tasks?task_ids=task-1&page_size=1',
    );
  });

  it.each([
    ['pipeline', { experiments: 1, pipelines: 0, recurringRuns: 1, runs: 2 }],
    ['experiment', { experiments: 0, pipelines: 1, recurringRuns: 1, runs: 2 }],
    ['two comparable runs', { experiments: 1, pipelines: 1, recurringRuns: 1, runs: 1 }],
    ['recurring run', { experiments: 1, pipelines: 1, recurringRuns: 0, runs: 2 }],
  ])('does not reuse a cluster missing its %s fixture', (_name, counts) => {
    expect(canReuseExistingData(counts)).toBe(false);
  });

  it('reuses existing data only when every capture fixture exists', () => {
    expect(canReuseExistingData({ experiments: 1, pipelines: 1, recurringRuns: 1, runs: 2 })).toBe(
      true,
    );
  });

  it('selects only V2 runs for comparison reuse', () => {
    expect(
      selectComparableRunIds([
        { run_id: 'v1', pipeline_spec: { workflow_manifest: '{}' } },
        {
          run_id: 'v2-a',
          pipeline_spec: { root: {}, schema_version: '2.1.0' },
        },
        {
          runId: 'v2-b',
          pipelineSpec: { root: {}, schemaVersion: '2.1.0' },
        },
        {
          run_id: 'legacy-wrapper',
          pipeline_spec: { pipeline_manifest: '{}' },
        },
        { run_id: 'missing-spec' },
      ]),
    ).toEqual(['v2-a', 'v2-b']);
  });

  it('continues through older V1 run pages until two V2 runs are found', async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce({
        next_page_token: 'page two',
        runs: Array.from({ length: 20 }, (_, index) => ({
          pipeline_spec: { workflow_manifest: '{}' },
          run_id: `v1-${index}`,
        })),
      })
      .mockResolvedValueOnce({
        runs: [
          { pipeline_spec: { root: {}, schema_version: '2.1.0' }, run_id: 'v2-a' },
          { pipeline_spec: { root: {}, schema_version: '2.1.0' }, run_id: 'v2-b' },
        ],
      });

    await expect(listComparableRunIds(request)).resolves.toEqual(['v2-a', 'v2-b']);
    expect(request).toHaveBeenNthCalledWith(1, 'GET', '/apis/v2beta1/runs?page_size=20');
    expect(request).toHaveBeenNthCalledWith(
      2,
      'GET',
      '/apis/v2beta1/runs?page_size=20&page_token=page%20two',
    );
  });

  it('returns the compatible runs found when pagination is exhausted', async () => {
    const request = vi.fn().mockResolvedValue({
      runs: [{ pipeline_spec: { root: {}, schema_version: '2.1.0' }, run_id: 'only-v2' }],
    });

    await expect(listComparableRunIds(request)).resolves.toEqual(['only-v2']);
    expect(request).toHaveBeenCalledTimes(1);
  });
});
