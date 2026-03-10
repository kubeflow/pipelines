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
import { vi } from 'vitest';
import { PipelineSpecTabContent } from './PipelineSpecTabContent';

// Mock Editor to capture the value prop passed to it
vi.mock('./Editor', () => ({
  __esModule: true,
  default: (props: { value: string; mode: string; readOnly: boolean }) => (
    <pre data-testid='editor-mock' data-mode={props.mode} data-readonly={props.readOnly}>
      {props.value}
    </pre>
  ),
}));

describe('PipelineSpecTabContent', () => {
  it('converts JSON template to YAML and passes to editor', () => {
    const jsonTemplate = JSON.stringify({ key: 'value', nested: { a: 1 } });
    const { getByTestId } = render(<PipelineSpecTabContent templateString={jsonTemplate} />);
    const editor = getByTestId('editor-mock');
    expect(editor.textContent).toContain('key: value');
    expect(editor.textContent).toContain('nested:');
    expect(editor.textContent).toContain('a: 1');
  });

  it('preserves YAML template format through round-trip', () => {
    const yamlTemplate = 'key: value\nnested:\n  a: 1\n';
    const { getByTestId } = render(<PipelineSpecTabContent templateString={yamlTemplate} />);
    const editor = getByTestId('editor-mock');
    expect(editor.textContent).toContain('key: value');
    expect(editor.textContent).toContain('nested:');
  });

  it('passes yaml mode and readOnly to the editor', () => {
    const { getByTestId } = render(<PipelineSpecTabContent templateString='name: test' />);
    const editor = getByTestId('editor-mock');
    expect(editor).toHaveAttribute('data-mode', 'yaml');
    expect(editor).toHaveAttribute('data-readonly', 'true');
  });
});
