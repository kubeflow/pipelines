/*
 * Copyright 2019 Google LLC
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
import { ViewerConfig } from './Viewer';
import { cssRaw } from 'typestyle';
import Markdown from 'markdown-to-jsx';

cssRaw(`
.markdown-viewer h1,
.markdown-viewer h2,
.markdown-viewer h3,
.markdown-viewer h4,
.markdown-viewer h5,
.markdown-viewer h6 {
position: relative;
  margin-top: 1em;
  margin-bottom: 16px;
  font-size: initial;
  font-weight: 700;
  line-height: 1.4;
}

.markdown-viewer h1,
.markdown-viewer h2 {
  padding-bottom: .3em;
  border-bottom: 1px solid #eee;
}

.markdown-viewer code,
.markdown-viewer pre {
  background-color: #f8f8f8;
  border-radius: 5px;
}

.markdown-viewer pre {
  border: solid 1px #eee;
  margin: 7px 0;
  padding: 7px;
}
`);

export interface MarkdownViewerConfig extends ViewerConfig {
  markdownContent: string;
}

interface MarkdownViewerProps {
  configs: MarkdownViewerConfig[];
  maxDimension?: number;
}

class MarkdownViewer extends React.Component<MarkdownViewerProps, any> {
  private _config = this.props.configs[0];
  private _props: MarkdownViewerProps;

  public render(): JSX.Element | null {
    if (!this._config) {
      return null;
    }
    return (
      <div className='markdown-viewer'>
        <Markdown>{this._config.markdownContent}</Markdown>
      </div>
    );
  }
}

export default MarkdownViewer;
