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
import { render } from '@testing-library/react';
import HTMLViewer, { HTMLViewerConfig } from './HTMLViewer';
import { PlotType } from './Viewer';
import TestUtils from '../../TestUtils';

describe('HTMLViewer', () => {
  it('does not break on empty data', () => {
    const { asFragment } = render(<HTMLViewer configs={[]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  const html = '<html><body><div>Hello World!</div></body></html>';
  const config: HTMLViewerConfig = {
    htmlContent: html,
    type: PlotType.WEB_APP,
  };

  it('renders some basic HTML', () => {
    const { asFragment } = render(<HTMLViewer configs={[config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a smaller snapshot version', () => {
    const { asFragment } = render(<HTMLViewer configs={[config]} maxDimension={100} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('uses srcdoc to insert HTML into the iframe', async () => {
    const { container } = render(<HTMLViewer configs={[config]} />);
    await TestUtils.flushPromises();
    const iframe = container.querySelector('iframe') as HTMLIFrameElement | null;
    expect(iframe).not.toBeNull();
    expect(iframe?.srcdoc).toEqual(html);
    expect(iframe?.src).toEqual('about:blank');
  });

  it('cannot be accessed from main frame of the other way around (no allow-same-origin)', async () => {
    const { container } = render(<HTMLViewer configs={[config]} />);
    await TestUtils.flushPromises();
    const iframe = container.querySelector('iframe') as HTMLIFrameElement | null;
    expect(iframe).not.toBeNull();
    expect((iframe as any)?.window).toBeUndefined();
    expect((iframe as any)?.document).toBeUndefined();
  });

  it('returns a user friendly display name', () => {
    expect(HTMLViewer.prototype.getDisplayName()).toBe('Static HTML');
  });
});
