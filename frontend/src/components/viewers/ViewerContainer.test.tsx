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
import ViewerContainer from './ViewerContainer';
import { PlotType } from './Viewer';
import { stableMuiSnapshotFragment } from 'src/testUtils/muiSnapshot';

describe('ViewerContainer', () => {
  const sampleConfigs: Record<PlotType, any> = {
    [PlotType.CONFUSION_MATRIX]: {
      type: PlotType.CONFUSION_MATRIX,
      data: [[1]],
      axes: ['x', 'y'],
      labels: ['label'],
    },
    [PlotType.MARKDOWN]: {
      type: PlotType.MARKDOWN,
      markdownContent: '# Title',
    },
    [PlotType.ROC]: {
      type: PlotType.ROC,
      data: [{ x: 0, y: 0, label: '0' }],
    },
    [PlotType.TABLE]: {
      type: PlotType.TABLE,
      data: [['cell']],
      labels: ['col'],
    },
    [PlotType.TENSORBOARD]: {
      type: PlotType.TENSORBOARD,
      url: 'http://test/url',
      namespace: 'test-ns',
    },
    [PlotType.VISUALIZATION_CREATOR]: {
      type: PlotType.VISUALIZATION_CREATOR,
    },
    [PlotType.WEB_APP]: {
      type: PlotType.WEB_APP,
      htmlContent: '<div>test</div>',
    },
  };

  it('does not break on empty configs', () => {
    const { container } = render(<ViewerContainer configs={[]} />);
    expect(container.firstChild).toBeNull();
  });

  Object.keys(PlotType).map(type =>
    it('renders a viewer of type ' + type, () => {
      const plotType = PlotType[type as keyof typeof PlotType];
      const { asFragment } = render(<ViewerContainer configs={[sampleConfigs[plotType]]} />);
      expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
    }),
  );
});
