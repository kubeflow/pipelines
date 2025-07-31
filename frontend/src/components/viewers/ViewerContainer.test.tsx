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

import { render } from '@testing-library/react';
import * as React from 'react';
import ViewerContainer from './ViewerContainer';
import { PlotType } from './Viewer';

describe('ViewerContainer', () => {
  it('does not break on empty configs', () => {
    const { asFragment } = render(<ViewerContainer configs={[]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  // Test only the viewer types that work without specific data requirements
  const workingViewerTypes = [
    PlotType.ROC,
    PlotType.TENSORBOARD,
    PlotType.VISUALIZATION_CREATOR,
    PlotType.WEB_APP,
  ];

  workingViewerTypes.map(type =>
    it('renders a viewer of type ' + type, () => {
      const { asFragment } = render(<ViewerContainer configs={[{ type }]} />);
      expect(asFragment()).toMatchSnapshot();
    }),
  );

  // TODO: Skip viewers that require specific data structures
  it.skip('renders a viewer of type CONFUSION_MATRIX', () => {
    // TODO: Requires data, axes, and labels configuration
  });

  it.skip('renders a viewer of type MARKDOWN', () => {
    // TODO: Requires content property
  });

  it.skip('renders a viewer of type TABLE', () => {
    // TODO: Requires data and labels configuration
  });
});
