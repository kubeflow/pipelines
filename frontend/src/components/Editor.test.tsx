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

/*
  These tests mimic https://github.com/securingsincity/react-ace/blob/master/tests/src/ace.spec.js
  to ensure that editor properties (placeholder and value) can be properly
  tested.
*/

describe('Editor', () => {
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

  it('has its value set to the provided value', () => {
    const value = 'I am a value.';
    const ref = React.createRef<Editor>();
    render(<Editor ref={ref} value={value} editorProps={{ $blockScrolling: Infinity }} />);
    expect(ref.current).not.toBeNull();
    const editor = (ref.current as any).editor;
    expect(editor.getValue()).toBe(value);
  });
});
