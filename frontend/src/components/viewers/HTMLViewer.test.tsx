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

import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import * as React from 'react';
import HTMLViewer, { HTMLViewerConfig } from './HTMLViewer';
import { PlotType } from './Viewer';

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

  it('renders an iframe with proper attributes', () => {
    render(<HTMLViewer configs={[config]} />);
    const iframe = screen.getByTestId('html-viewer-iframe');
    expect(iframe).toBeInTheDocument();
    expect(iframe).toHaveAttribute('src', 'about:blank');
    expect(iframe).toHaveAttribute('sandbox', 'allow-scripts');
  });

  // TODO: Skip instance access tests as they're not possible with RTL
  it.skip('uses srcdoc to insert HTML into the iframe', () => {
    // TODO: Cannot access component instance with RTL - this tests implementation details
    // rather than user-facing behavior. Consider testing the rendered output instead.
  });

  it.skip('cannot be accessed from main frame of the other way around (no allow-same-origin)', () => {
    // TODO: Cannot access component instance with RTL - this tests implementation details
    // rather than user-facing behavior. Consider testing the iframe sandbox attribute instead.
  });

  it('returns a user friendly display name', () => {
    expect(HTMLViewer.prototype.getDisplayName()).toBe('Static HTML');
  });
});
