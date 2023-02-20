/*
 * Copyright 2023 The Kubeflow Authors
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
import { classes } from 'typestyle';
import MD2Tabs from '../atoms/MD2Tabs';
import { PageProps } from './Page';
import PipelineList from './PipelineList';
import { RoutePage } from '../components/Router';
import { NamespaceContext } from '../lib/KubeflowClient';
import { commonCss, padding } from '../Css';
import { BuildInfoContext } from 'src/lib/BuildInfo';

export enum PrivateAndSharedTab {
  PRIVATE = 0,
  SHARED = 1,
}

export interface PrivateAndSharedProps extends PageProps {
  view: PrivateAndSharedTab;
}

const PrivatePipelineList: React.FC<PageProps> = props => {
  const namespace = React.useContext(NamespaceContext);
  return <PipelineList {...props} namespace={namespace} />;
};

export enum PipelineTabsHeaders {
  PRIVATE = 'Private',
  SHARED = 'Shared',
}

export enum PipelineTabsTooltips {
  PRIVATE = 'Only people who have access to this namespace will be able to view and use these pipelines.',
  SHARED = 'Everyone in your organization will be able to view and use these pipelines.',
}

const PrivateAndSharedPipelines: React.FC<PrivateAndSharedProps> = props => {
  const buildInfo = React.useContext(BuildInfoContext);

  const tabSwitched = (newTab: PrivateAndSharedTab): void => {
    props.history.push(
      newTab === PrivateAndSharedTab.PRIVATE ? RoutePage.PIPELINES : RoutePage.PIPELINES_SHARED,
    );
  };

  if (!buildInfo?.apiServerMultiUser) {
    return <PipelineList {...props} />;
  }
  return (
    <div className={classes(commonCss.page, padding(20, 't'))}>
      <MD2Tabs
        tabs={[
          {
            header: PipelineTabsHeaders.PRIVATE,
            tooltip: PipelineTabsTooltips.PRIVATE,
          },
          {
            header: PipelineTabsHeaders.SHARED,
            tooltip: PipelineTabsTooltips.SHARED,
          },
        ]}
        selectedTab={props.view}
        onSwitch={tabSwitched}
      />

      {props.view === PrivateAndSharedTab.PRIVATE && <PrivatePipelineList {...props} />}

      {props.view === PrivateAndSharedTab.SHARED && <PipelineList {...props} />}
    </div>
  );
};

export default PrivateAndSharedPipelines;
