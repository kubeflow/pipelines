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
import Viewer, { ViewerConfig } from './Viewer';
import { cssRaw } from 'typestyle';
import Markdown from 'markdown-to-jsx';
import Banner from '../Banner';
import { ExternalLink } from 'src/atoms/ExternalLink';

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

const MAX_MARKDOWN_STR_LENGTH = 50 * 1000 * 8; // 50KB

export interface MarkdownViewerConfig extends ViewerConfig {
  markdownContent: string;
}

export interface MarkdownViewerProps {
  configs: MarkdownViewerConfig[];
  maxDimension?: number;
  maxLength?: number;
}

class MarkdownViewer extends Viewer<MarkdownViewerProps, any> {
  public getDisplayName(): string {
    return 'Markdown';
  }

  public render(): JSX.Element | null {
    if (!this.props.configs[0]) {
      return null;
    }
    return (
      <div className='markdown-viewer'>
        <MarkdownAdvanced
          maxMarkdownStrLength={this.props.maxLength}
          content={this.props.configs[0].markdownContent}
        />
      </div>
    );
  }
}

export default MarkdownViewer;

interface MarkdownAdvancedProps {
  maxMarkdownStrLength?: number;
  content: string;
}

function preventEventBubbling(e: React.MouseEvent): void {
  e.stopPropagation();
}
const renderExternalLink = (props: {}) => (
  <ExternalLink {...props} onClick={preventEventBubbling} />
);
const markdownOptions = {
  overrides: { a: { component: renderExternalLink } },
  disableParsingRawHTML: true,
};

const MarkdownAdvanced = ({
  maxMarkdownStrLength = MAX_MARKDOWN_STR_LENGTH,
  content,
}: MarkdownAdvancedProps) => {
  // truncatedContent will be memoized, each call with the same content + maxMarkdownStrLength arguments will return the same truncatedContent without calculation.
  // Reference: https://reactjs.org/docs/hooks-reference.html#usememo
  const truncatedContent = React.useMemo(() => content.substr(0, maxMarkdownStrLength), [
    maxMarkdownStrLength,
    content,
  ]);

  return (
    <>
      {content.length > maxMarkdownStrLength && (
        <Banner message='This markdown is too large to render completely.' mode={'warning'} />
      )}
      <Markdown options={markdownOptions}>{truncatedContent}</Markdown>
    </>
  );
};
