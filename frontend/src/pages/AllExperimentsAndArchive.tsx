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
import { useNavigate } from 'react-router-dom';
import ExperimentList from './ExperimentList';
import ArchivedExperiments from './ArchivedExperiments';
import MD2Tabs from '../atoms/MD2Tabs';
import { RoutePage } from '../components/Router';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { PageProps } from './Page';

export enum AllExperimentsAndArchiveTab {
  EXPERIMENTS = 0,
  ARCHIVE = 1,
}

export interface AllExperimentsAndArchiveProps extends PageProps {
  view: AllExperimentsAndArchiveTab;
  namespace?: string;
  onError?: (message: string, error: Error) => void;
}

const AllExperimentsAndArchive: React.FC<AllExperimentsAndArchiveProps> = props => {
  const navigate = useNavigate();

  const tabSwitched = (newTab: AllExperimentsAndArchiveTab): void => {
    const path =
      newTab === AllExperimentsAndArchiveTab.EXPERIMENTS
        ? RoutePage.EXPERIMENTS
        : RoutePage.ARCHIVED_EXPERIMENTS;
    navigate(path);
  };

  return (
    <div className={classes(commonCss.page, padding(20, 't'))}>
      <MD2Tabs tabs={['Active', 'Archived']} selectedTab={props.view} onSwitch={tabSwitched} />
      {props.view === 0 && <ExperimentList {...props} />}
      {props.view === 1 && <ArchivedExperiments {...props} />}
    </div>
  );
};

export default AllExperimentsAndArchive;
