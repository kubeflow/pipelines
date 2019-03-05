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
import Viewer, { ViewerConfig } from './Viewer';
// tslint:disable-next-line:no-var-requires
const markdownIt = require('markdown-it');

export interface MarkdownViewerConfig extends ViewerConfig {
  markdownContent: string;
}

interface MarkdownViewerProps {
  configs: MarkdownViewerConfig[];
  maxDimension?: number;
}

class MarkdownViewer extends Viewer<MarkdownViewerProps, any> {
  private _config = this.props.configs[0];

  constructor(props: any) {
    super(props);
  }

  public getDisplayName(): string {
    return 'Markdown';
  }

  public render(): JSX.Element | null {
    if (!this._config) {
      return null;
    }
    const html = markdownIt().render(this._config.markdownContent);
    return <div dangerouslySetInnerHTML={{ __html: html }} />;
  }
}

export default MarkdownViewer;
