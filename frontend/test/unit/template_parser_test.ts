import * as assert from 'assert';
import * as fs from 'fs';
import 'jasmine';
import * as TemplateParser from '../../src/lib/template_parser';

describe('template parser', () => {
  it('parses output paths from correct mock template', () => {
    const mockTemplate = fs.readFileSync('../mock-backend/mock-template.yaml', 'utf-8');
    const baseOutputPath = 'gs://test/output/path';
    const jobId = 'test-job-id';

    const expectedPaths = [
      `${baseOutputPath}/${jobId}/analysis`,
      `${baseOutputPath}/${jobId}/transform`,
      `${baseOutputPath}/${jobId}/model`
    ];
    const paths = TemplateParser.parseTemplateOuputPaths(mockTemplate, baseOutputPath, jobId);
    assert.deepEqual(paths, expectedPaths);
  });

  it('throws an error if spec has no entrypoint', () => {
    const mockTemplate = 'bad yaml';
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        /Spec does not contain an entrypoint/);
  });

  it('throws an error if spec has no templates', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = `
      spec:
        entrypoint: ${entrypoint}
    `;
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        /Spec does not contain any templates/);
  });

  it('throws an error if spec has no template for the entrypoint', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = `
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: template1
    `;
    assert.throws(() => TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''),
        new RegExp('Could not find template for entrypoint: ' + entrypoint));
  });

  it('returns an empty array if entrypoint has no steps', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = `
      spec:
        entrypoint: ${entrypoint}
        templates:
        - name: ${entrypoint}
    `;
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), []);
  });

  it('does not return an output for steps with no output parameter', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = `
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
    `;
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), []);
  });

  it('extracts outputs for steps running in parallel', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = `
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
    `;
    const expectedPaths = [
      'output/path1',
      'output/path2',
    ];
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), expectedPaths);
  });

  it('extracts outputs for steps running sequentially', () => {
    const entrypoint = 'entrypoint-name';
    const mockTemplate = `
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
    `;
    const expectedPaths = [
      'output/path1',
      'output/path2',
    ];
    assert.deepEqual(TemplateParser.parseTemplateOuputPaths(mockTemplate, '', ''), expectedPaths);
  });
});
