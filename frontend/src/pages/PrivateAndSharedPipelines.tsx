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
import { InspectionTabs } from 'src/components/modernization/InspectionTabs';
import 'src/components/modernization/Pipelines.css';
import { PageProps } from './Page';
import PipelineList from './PipelineList';
import { RoutePage } from '../components/Router';
import { NamespaceContext } from '../lib/KubeflowClient';
import { BuildInfoContext } from 'src/lib/BuildInfo';

export enum PrivateAndSharedTab {
  PRIVATE = 0,
  SHARED = 1,
}

export interface PrivateAndSharedProps extends PageProps {
  view: PrivateAndSharedTab;
}

const PrivatePipelineList: React.FC<PageProps> = (props) => {
  const namespace = React.useContext(NamespaceContext);
  return <PipelineList key={namespace || 'private'} {...props} namespace={namespace} />;
};

export enum PipelineTabsHeaders {
  PRIVATE = 'Private',
  SHARED = 'Shared',
}

export enum PipelineTabsTooltips {
  PRIVATE = 'Only people who have access to this namespace will be able to view and use these pipelines.',
  SHARED = 'Everyone in your organization will be able to view and use these pipelines.',
}

const PrivateAndSharedPipelines: React.FC<PrivateAndSharedProps> = (props) => {
  const buildInfo = React.useContext(BuildInfoContext);

  const tabSwitched = (newTab: PrivateAndSharedTab): void => {
    props.navigate(
      newTab === PrivateAndSharedTab.PRIVATE ? RoutePage.PIPELINES : RoutePage.PIPELINES_SHARED,
    );
  };

  if (!buildInfo?.apiServerMultiUser) {
    return <PipelineList {...props} />;
  }
  return (
    <InspectionTabs
      className='kfp-pipelines-tabs'
      ariaLabel='Pipeline visibility'
      tabs={[PipelineTabsHeaders.PRIVATE, PipelineTabsHeaders.SHARED]}
      selectedTab={props.view}
      onSwitch={tabSwitched}
    >
      <p className='kfp-pipelines-scope'>
        {props.view === PrivateAndSharedTab.PRIVATE
          ? PipelineTabsTooltips.PRIVATE
          : PipelineTabsTooltips.SHARED}
      </p>
      {props.view === PrivateAndSharedTab.PRIVATE ? (
        <PrivatePipelineList {...props} />
      ) : (
        <PipelineList {...props} />
      )}
    </InspectionTabs>
  );
};

export default PrivateAndSharedPipelines;
