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

import React, { ComponentType } from 'react';
import ConfusionMatrix from './ConfusionMatrix';
import HTMLViewer from './HTMLViewer';
import MarkdownViewer from './MarkdownViewer';
import PagedTable from './PagedTable';
import ROCCurve from './ROCCurve';
import TensorboardViewer from './Tensorboard';
import { PlotType, ViewerConfig } from './Viewer';
import VisualizationCreator from './VisualizationCreator';

type ViewerProperties = {
  component: ComponentType<any>;
  isAggregatable: boolean;
  displayNameKey: string;
};

export const componentMap: Record<PlotType, ViewerProperties> = {
  [PlotType.CONFUSION_MATRIX]: {
    component: ConfusionMatrix,
    isAggregatable: false,
    displayNameKey: 'common:confusionMatrix',
  },
  [PlotType.MARKDOWN]: {
    component: MarkdownViewer,
    isAggregatable: false,
    displayNameKey: 'common:markdown',
  },
  [PlotType.ROC]: {
    component: ROCCurve,
    isAggregatable: true,
    displayNameKey: 'common:rocCurve',
  },
  [PlotType.TABLE]: {
    component: PagedTable,
    isAggregatable: false,
    displayNameKey: 'common:table',
  },
  [PlotType.TENSORBOARD]: {
    component: TensorboardViewer,
    isAggregatable: true,
    displayNameKey: 'common:tensorboard',
  },
  [PlotType.VISUALIZATION_CREATOR]: {
    component: VisualizationCreator,
    isAggregatable: false,
    displayNameKey: 'common:VisualizationCreator',
  },
  [PlotType.WEB_APP]: {
    component: HTMLViewer,
    isAggregatable: false,
    displayNameKey: 'common:staticHtml',
  },
};

interface ViewerContainerProps {
  configs: ViewerConfig[];
  maxDimension?: number;
}

class ViewerContainer extends React.Component<ViewerContainerProps> {
  public render(): JSX.Element | null {
    const { configs, maxDimension } = this.props;
    if (!configs.length) {
      return null;
    }

    const Component = componentMap[configs[0].type].component;
    return <Component configs={configs as any} maxDimension={maxDimension} />;
  }
}

export default ViewerContainer;
