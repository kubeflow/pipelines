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

import AceEditor from 'react-ace';

// Modified AceEditor that supports HTML within provided placeholder. This is
// important because it allows for the usage of multi-line placeholders.
class Editor extends AceEditor {
  public static defaultProps = {
    ...AceEditor.defaultProps,
    editorProps: {
      ...(AceEditor.defaultProps?.editorProps || {}),
      $blockScrolling: Infinity,
    },
  };

  public componentDidMount(): void {
    super.componentDidMount();
    if (this.editor) {
      (this.editor as { $blockScrolling?: number }).$blockScrolling = Infinity;
    }
  }

  public updatePlaceholder(): void {
    const editor = this.editor;
    const placeholder = this.props.placeholder ?? '';
    const renderer = editor.renderer as {
      placeholderNode?: HTMLDivElement | null;
      scroller: HTMLElement;
    };

    const showPlaceholder = !editor.session.getValue().length;
    const existingNode = renderer.placeholderNode ?? null;
    if (!showPlaceholder && existingNode) {
      renderer.scroller.removeChild(existingNode);
      renderer.placeholderNode = null;
    } else if (showPlaceholder && !existingNode) {
      const node = document.createElement('div');
      node.innerHTML = placeholder;
      node.className = 'ace_comment ace_placeholder';
      node.style.padding = '0 9px';
      node.style.position = 'absolute';
      node.style.zIndex = '3';
      renderer.placeholderNode = node;
      renderer.scroller.appendChild(node);
    } else if (showPlaceholder && existingNode) {
      existingNode.innerHTML = placeholder;
    }
  }
}

export default Editor;
