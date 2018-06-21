import * as assert from 'assert';
import * as fs from 'fs';
import 'jasmine';
import { PackageTemplate } from '../../src/api/pipeline_package';
import * as TemplateParser from '../../src/lib/template_parser';

function mockApiResponseForTemplate(mockTemplate: string): PackageTemplate {
  const packageTemplate: PackageTemplate = {
    template: mockTemplate,
  };
  return packageTemplate;
}

describe('template parser', () => {
  it('parses output paths from correct mock template', () => {
    const mockTemplate = mockApiResponseForTemplate(fs.readFileSync('../mock-backend/mock-template.yaml', 'utf-8'));
    const baseOutputPath = 'gs://test/output/path';
    const jobId = 'test-job-id';

    const expectedPaths: TemplateParser.OutputInfo[] = [
      { path: `${baseOutputPath}/${jobId}/analysis`, step: 'analyze' },
      { path: `${baseOutputPath}/${jobId}/transform`, step: 'transform' },
      { path: `${baseOutputPath}/${jobId}/model`, step: 'train' },
    ];
    const paths = TemplateParser.parseTemplateOuputPaths(mockTemplate, baseOutputPath, jobId);
    assert.deepStrictEqual(paths, expectedPaths);
  });

  it('throws an error if spec has no entrypoint', () => {
    const mockTemplate = mockApiResponseForTemplate('bad yaml');
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        /Workflow argo template does not contain a spec/);
  });

  it('throws an error if spec has no entrypoint', () => {
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        templates:
        - name: template1
    `);
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        /Spec does not contain an entrypoint/);
  });

  it('throws an error if spec has no templates', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        entrypoint: ${entrypoint}
    `);
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        /Spec does not contain any templates/);
  });

  it('throws an error if spec has no template for the entrypoint', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: template1
    `);
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        new RegExp('Could not find template for entrypoint: ' + entrypoint));
  });

  it('returns an empty array if entrypoint has no steps', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: ${entrypoint}
    `);
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), []);
  });

  it('does not return an output for steps with no output parameter', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: ${entrypoint}
          steps:
          - name: step1
            arguments:
              parameters:
              - name: not-output
                value: not-output-path
    `);
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), []);
  });

  it('extracts outputs for steps running in parallel', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: ${entrypoint}
          steps:
          - - name: step1
              arguments:
                parameters:
                - name: output
                  value: output/path1
            - name: step2
              arguments:
                parameters:
                - name: output
                  value: output/path2
    `);
    const expectedPaths = [
      { path: 'output/path1', step: 'step1' },
      { path: 'output/path2', step: 'step2' },
    ];
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), expectedPaths);
  });

  it('extracts outputs for steps running sequentially', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = mockApiResponseForTemplate(`
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: ${entrypoint}
          steps:
          - name: step1
            arguments:
              parameters:
              - name: output
                value: output/path1
          - - name: step2
              arguments:
                parameters:
                - name: output
                  value: output/path2
    `);
    const expectedPaths = [
      { path: 'output/path1', step: 'step1' },
      { path: 'output/path2', step: 'step2' },
    ];
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), expectedPaths);
  });
});
