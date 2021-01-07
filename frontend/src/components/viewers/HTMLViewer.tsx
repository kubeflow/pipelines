/*
 * Copyright 2018 Google LLC
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
import { color } from '../../Css';
import { stylesheet } from 'typestyle';

export interface HTMLViewerConfig extends ViewerConfig {
  htmlContent: string;
}

interface HTMLViewerProps {
  configs: HTMLViewerConfig[];
  maxDimension?: number;
}

class HTMLViewer extends React.Component<HTMLViewerProps, any> {
  private _iframeRef = React.createRef<HTMLIFrameElement>();
  private _config = this.props.configs[0];

  private _css = stylesheet({
    iframe: {
      border: '1px solid ' + color.divider,
      boxSizing: 'border-box',
      flexGrow: 1,
      height: this.props.maxDimension ? this.props.maxDimension : 'initial',
      minHeight: this.props.maxDimension ? this.props.maxDimension : 600,
      width: this.props.maxDimension ? this.props.maxDimension : '100%',
    },
  });

  public componentDidMount(): void {
    // TODO: iframe.srcdoc doesn't work on Edge yet. It's been added, but not
    // yet rolled out as of the time of writing this (6/14/18):
    // https://developer.microsoft.com/en-us/microsoft-edge/platform/issues/12375527/
    // I'm using this since it seems like the safest way to insert HTML into an
    // iframe, while allowing Javascript, but without needing to require
    // "allow-same-origin" sandbox rule.
    if (this._iframeRef.current) {
      this._iframeRef.current!.srcdoc = this._config.htmlContent;
    }
  }

  public render(): JSX.Element | null {
    if (!this._config) {
      return null;
    }

    return (
      // TODO: fix this
      // eslint-disable-next-line jsx-a11y/iframe-has-title
      <iframe
        ref={this._iframeRef}
        src='about:blank'
        className={this._css.iframe}
        sandbox='allow-scripts'
      />
    );
  }
}
export default HTMLViewer;
