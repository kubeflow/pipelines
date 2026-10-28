/*
 * Copyright 2018 The Kubeflow Authors
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
import './RichViewers.css';

export interface HTMLViewerConfig extends ViewerConfig {
  htmlContent: string;
}

interface HTMLViewerProps {
  configs: HTMLViewerConfig[];
  maxDimension?: number;
}

class HTMLViewer extends Viewer<HTMLViewerProps, any> {
  public getDisplayName(): string {
    return 'Static HTML';
  }

  public render(): React.JSX.Element | null {
    const config = this.props.configs[0];
    if (!config) {
      return null;
    }

    return (
      <iframe
        title='HTML report'
        srcDoc={config.htmlContent}
        src='about:blank'
        className='kfp-html-viewer'
        style={{ height: this.props.maxDimension, minHeight: this.props.maxDimension || 600 }}
        sandbox='allow-scripts'
      />
    );
  }
}

export default HTMLViewer;
