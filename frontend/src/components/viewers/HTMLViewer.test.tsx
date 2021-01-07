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
import { mount } from 'enzyme';
import HTMLViewer, { HTMLViewerConfig } from './HTMLViewer';
import { PlotType } from './Viewer';
import { TFunction } from 'i18next';
import { componentMap } from './ViewerContainer';

describe('HTMLViewer', () => {
  let t: TFunction = (key: string) => key;
  it('does not break on empty data', () => {
    const tree = mount(<HTMLViewer configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  const html = '<html><body><div>Hello World!</div></body></html>';
  const config: HTMLViewerConfig = {
    htmlContent: html,
    type: PlotType.WEB_APP,
  };

  it('renders some basic HTML', () => {
    const tree = mount(<HTMLViewer configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a smaller snapshot version', () => {
    const tree = mount(<HTMLViewer configs={[config]} maxDimension={100} />);
    expect(tree).toMatchSnapshot();
  });

  it('uses srcdoc to insert HTML into the iframe', () => {
    const tree = mount(<HTMLViewer configs={[config]} />);
    expect((tree.instance() as any)._iframeRef.current.srcdoc).toEqual(html);
    expect((tree.instance() as any)._iframeRef.current.src).toEqual('about:blank');
  });

  it('cannot be accessed from main frame of the other way around (no allow-same-origin)', () => {
    const tree = mount(<HTMLViewer configs={[config]} />);
    expect((tree.instance() as any)._iframeRef.current.window).toBeUndefined();
    expect((tree.instance() as any)._iframeRef.current.document).toBeUndefined();
  });

  it('returns a user friendly display name', () => {
    expect((componentMap[PlotType.WEB_APP].displayNameKey = 'common:staticHtml'));
  });
});
