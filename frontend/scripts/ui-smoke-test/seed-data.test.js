const {
  createRecurringRun,
  createRun,
  PIPELINE_YAML_BY_PROFILE,
  uploadPipeline,
  uploadPipelineVersion,
} = require('./seed-data.js');
const YAML = require('yaml');

describe('UI smoke native v2 data seeding', () => {
  it('uploads PipelineSpec IR so pipeline detail captures have a stored version', async () => {
    const request = vi.fn().mockResolvedValue({ pipeline_id: 'pipeline-1' });
    await uploadPipeline('smoke-pipeline', 'Smoke description', request);
    const [method, endpoint, body, options] = request.mock.calls[0];
    expect(method).toBe('POST');
    expect(endpoint).toMatch(/^\/apis\/v2beta1\/pipelines\/upload\?/);
    expect(body).toBeNull();
    expect(options.rawBody.toString()).toContain('schemaVersion: 2.1.0');
    expect(options.headers['Content-Type'] || options.headers['content-type']).toContain(
      'multipart/form-data',
    );
  });

  it('uploads an explicit pipeline version through the v2 endpoint', async () => {
    const request = vi.fn().mockResolvedValue({ pipeline_version_id: 'version-1' });
    await uploadPipelineVersion('pipeline-1', request);
    expect(request.mock.calls[0][1]).toMatch(/^\/apis\/v2beta1\/pipelines\/upload_version\?/);
    expect(new URLSearchParams(request.mock.calls[0][1].split('?')[1]).get('pipelineid')).toBe(
      'pipeline-1',
    );
  });

  it.each(Object.entries(PIPELINE_YAML_BY_PROFILE))(
    'keeps the %s fixture in native PipelineSpec format',
    (_name, source) => {
      const spec = YAML.parse(source);
      expect(spec.schemaVersion).toBe('2.1.0');
      expect(spec.root.dag.tasks).toBeDefined();
      expect(spec.deploymentSpec.executors).toBeDefined();
      expect(spec).not.toHaveProperty('implementation');
      expect(spec).not.toHaveProperty('workflow_manifest');
    },
  );

  it.each([
    ['run', createRun, '/apis/v2beta1/runs', { run_id: 'run-1' }],
    [
      'schedule',
      createRecurringRun,
      '/apis/v2beta1/recurringruns',
      { recurring_run_id: 'schedule-1' },
    ],
  ])(
    'creates a %s referencing the uploaded v2 pipeline version',
    async (_name, create, endpoint, result) => {
      const request = vi.fn().mockResolvedValue(result);
      await create('Smoke fixture', 'pipeline-1', 'experiment-1', request, 'version-1');
      expect(request).toHaveBeenCalledWith(
        'POST',
        endpoint,
        expect.objectContaining({
          experiment_id: 'experiment-1',
          pipeline_version_reference: {
            pipeline_id: 'pipeline-1',
            pipeline_version_id: 'version-1',
          },
          runtime_config: { parameters: {} },
        }),
      );
    },
  );
});
