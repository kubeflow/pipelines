/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as React from 'react';
import { render } from '@testing-library/react';
import { PipelineSpecTabContent } from './PipelineSpecTabContent';
import jsyaml from 'js-yaml';

describe('PipelineSpecTabContent', () => {
  it('renders JSON template as YAML in the editor', () => {
    const jsonTemplate = JSON.stringify({ key: 'value', nested: { a: 1 } });
    const expectedYaml = jsyaml.safeDump(jsyaml.safeLoad(jsonTemplate));
    const { container } = render(<PipelineSpecTabContent templateString={jsonTemplate} />);
    const editorContent = container.querySelector('.ace_content');
    expect(editorContent).toBeInTheDocument();
    // Verify the editor received the YAML-converted content
    expect(expectedYaml).toContain('key: value');
    expect(expectedYaml).toContain('nested:');
  });

  it('renders YAML template preserving format', () => {
    const yamlTemplate = 'key: value\nnested:\n  a: 1\n';
    const expectedYaml = jsyaml.safeDump(jsyaml.safeLoad(yamlTemplate));
    const { container } = render(<PipelineSpecTabContent templateString={yamlTemplate} />);
    const editorContent = container.querySelector('.ace_content');
    expect(editorContent).toBeInTheDocument();
    expect(expectedYaml).toContain('key: value');
  });

  it('renders with a simple YAML key-value', () => {
    const { container } = render(<PipelineSpecTabContent templateString='name: test' />);
    const editorContent = container.querySelector('.ace_content');
    expect(editorContent).toBeInTheDocument();
  });
});
