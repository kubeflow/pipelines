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

// Extend the AceEditor types to include the internal editor object
interface ExtendedAceEditor extends AceEditor {
  editor: any; // Using any here as the Ace editor types are complex
}

// Modified AceEditor that supports HTML within provided placeholder. This is
// important because it allows for the usage of multi-line placeholders.
class Editor extends AceEditor {
  public updatePlaceholder(): void {
    const editor = (this as ExtendedAceEditor).editor;
    const { placeholder } = this.props;

    const showPlaceholder = !editor.session.getValue().length;
    let node = editor.renderer.placeholderNode as HTMLDivElement | null;
    
    if (!showPlaceholder && node) {
      if (editor.renderer.scroller.contains(node)) {
        editor.renderer.scroller.removeChild(node);
      }
      editor.renderer.placeholderNode = null;
    } else if (showPlaceholder && !node) {
      node = document.createElement('div');
      node.innerHTML = placeholder || '';
      node.className = 'ace_comment ace_placeholder';
      node.style.padding = '0 9px';
      node.style.position = 'absolute';
      node.style.zIndex = '3';
      editor.renderer.placeholderNode = node;
      editor.renderer.scroller.appendChild(node);
    } else if (showPlaceholder && node) {
      node.innerHTML = placeholder || '';
    }
  }
}

export default Editor;
