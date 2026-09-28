/*
 * Copyright 2019 The Kubeflow Authors
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
import Editor from './Editor';
import { config } from 'ace-builds';
import yamlWorkerUrl from 'ace-builds/src-noconflict/worker-yaml.js?url';
import jsonWorkerUrl from 'ace-builds/src-noconflict/worker-json.js?url';

/*
  These tests mimic https://github.com/securingsincity/react-ace/blob/master/tests/src/ace.spec.js
  to ensure that editor properties (placeholder and value) can be properly
  tested.
*/

describe('Editor', () => {
  it('resolves YAML and JSON workers to bundled asset URLs', () => {
    expect(config.moduleUrl('ace/mode/yaml_worker', 'worker')).toBe(yamlWorkerUrl);
    expect(config.moduleUrl('ace/mode/json_worker', 'worker')).toBe(jsonWorkerUrl);
  });

  // Ace renders a large, environment-dependent DOM tree. Snapshot tests are brittle
  // and create noisy diffs during dependency upgrades. Assert key behaviors instead.
  const getPlaceholderNode = (container: HTMLElement) =>
    container.querySelector('.ace_placeholder') as HTMLElement | null;

  it('renders without a placeholder and value', () => {
    const { container } = render(<Editor editorProps={{ $blockScrolling: Infinity }} />);
    expect(container.querySelector('.ace_editor')).not.toBeNull();
    expect(getPlaceholderNode(container)).toBeNull();
  });

  it('renders with a placeholder', () => {
    const placeholder = 'I am a placeholder.';
    const { container } = render(
      <Editor placeholder={placeholder} editorProps={{ $blockScrolling: Infinity }} />,
    );
    const placeholderNode = getPlaceholderNode(container);
    expect(placeholderNode).not.toBeNull();
    expect(placeholderNode?.textContent).toBe(placeholder);
  });

  it('renders a placeholder that contains HTML', () => {
    const placeholder = 'I am a placeholder with <strong>HTML</strong>.';
    const { container } = render(
      <Editor placeholder={placeholder} editorProps={{ $blockScrolling: Infinity }} />,
    );
    const placeholderNode = getPlaceholderNode(container);
    expect(placeholderNode).not.toBeNull();
    expect(placeholderNode?.innerHTML).toBe(placeholder);
  });

  it('keeps the same editor session and read-only value across palette changes', () => {
    const ref = React.createRef<Editor>();
    const renderEditor = (dark: boolean) => (
      <div className={`kfp-theme${dark ? ' dark' : ''}`}>
        <Editor ref={ref} value='name: example' mode='yaml' theme='github' readOnly={true} />
      </div>
    );
    const { container, rerender } = render(renderEditor(false));
    const editor = ref.current!.editor;
    editor.selection.moveCursorTo(0, 5);
    rerender(renderEditor(true));
    expect(ref.current!.editor).toBe(editor);
    expect(editor.getValue()).toBe('name: example');
    expect(editor.getReadOnly()).toBe(true);
    expect(editor.getCursorPosition()).toEqual({ row: 0, column: 5 });
    expect(container.querySelector('.ace_editor')).toHaveClass('ace-github');
  });

  it('has its value set to the provided value', () => {
    const value = 'I am a value.';
    const ref = React.createRef<Editor>();
    render(<Editor ref={ref} value={value} editorProps={{ $blockScrolling: Infinity }} />);
    expect(ref.current).not.toBeNull();
    const editor = (ref.current as any).editor;
    expect(editor.getValue()).toBe(value);
  });
});
