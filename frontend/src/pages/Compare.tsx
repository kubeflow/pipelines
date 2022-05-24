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

import React, { useState } from 'react';
import { Redirect } from 'react-router-dom';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { classes, stylesheet } from 'typestyle';
import { Workflow } from '../third_party/mlmd/argo_template';
import { ApiRunDetail } from '../apis/run';
import Hr from '../atoms/Hr';
import Separator from '../atoms/Separator';
import CollapseButton from '../components/CollapseButton';
import CompareTable, { CompareTableProps } from '../components/CompareTable';
import PlotCard, { PlotCardProps } from '../components/PlotCard';
import { QUERY_PARAMS, RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { componentMap } from '../components/viewers/ViewerContainer';
import { commonCss, padding } from '../Css';
import { Apis } from '../lib/Apis';
import Buttons from '../lib/Buttons';
import CompareUtils from '../lib/CompareUtils';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { URLParser } from '../lib/URLParser';
import { logger } from '../lib/Utils';
import WorkflowParser from '../lib/WorkflowParser';
import { Page, PageProps } from './Page';
import RunList from './RunList';
import CompareV1 from './CompareV1';
import CompareV2 from './CompareV2';
import { FeatureKey, isFeatureEnabled } from 'src/features';

const css = stylesheet({
  outputsRow: {
    marginLeft: 15,
    overflowX: 'auto',
  },
});

export interface TaggedViewerConfig {
  config: ViewerConfig;
  runId: string;
  runName: string;
}

export interface CompareState {
  collapseSections: { [key: string]: boolean };
  fullscreenViewerConfig: PlotCardProps | null;
  paramsCompareProps: CompareTableProps;
  metricsCompareProps: CompareTableProps;
  runs: ApiRunDetail[];
  selectedIds: string[];
  viewersMap: Map<PlotType, TaggedViewerConfig[]>;
  workflowObjects: Workflow[];
}

const overviewSectionName = 'Run overview';
const paramsSectionName = 'Parameters';
const metricsSectionName = 'Metrics';

function Compare(props: CompareState) {
  // const [collapseSections, setCollapseSections] = useState({});
  // const [fullscreenViewerConfig, setFullscreenViewerConfig] = useState(null);
  // const [metricsCompareProps, setMetricsCompareProps] = useState({ rows: [], xLabels: [], yLabels: [] });
  // const [paramsCompareProps, setParamsCompareProps] = useState({ rows: [], xLabels: [], yLabels: [] });
  // const [runs, setRuns] = useState([]);
  // const [selectedIds, setSelectedIds] = useState([]);
  // const [viewersMap, setViewersMap] = useState(new Map());
  // const [workflowObjects, setWorkflowObjects] = useState([]);

  const showV2Compare =
      isFeatureEnabled(FeatureKey.V2_ALPHA);

  if (!showV2Compare) {
    return (
      <CompareV1 {...props} />
    );
  } else {
    return (
      <CompareV2 {...props} />
    );
  }
}

export default Compare;
