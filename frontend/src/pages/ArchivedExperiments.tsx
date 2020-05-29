/*
 * Copyright 2020 Google LLC
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
import Buttons from '../lib/Buttons';
import ExperimentList from '../components/ExperimentList';
import { Page, PageProps } from './Page';
import { ExperimentStorageState } from '../apis/experiment';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { NamespaceContext } from 'src/lib/KubeflowClient';

interface ArchivedExperimentsProp {
  namespace?: string;
}

interface ArchivedExperimentsState {}

export class ArchivedExperiments extends Page<ArchivedExperimentsProp, ArchivedExperimentsState> {
  private _experimentlistRef = React.createRef<ExperimentList>();

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons.refresh(this.refresh.bind(this)).getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Archive',
    };
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <ExperimentList
          onError={this.showPageError.bind(this)}
          ref={this._experimentlistRef}
          storageState={ExperimentStorageState.ARCHIVED}
          {...this.props}
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    // Tell run list to refresh
    if (this._experimentlistRef.current) {
      this.clearBanner();
      await this._experimentlistRef.current.refresh();
    }
  }
}

const EnhancedArchivedExperiments = (props: PageProps) => {
  const namespace = React.useContext(NamespaceContext);
  return <ArchivedExperiments key={namespace} {...props} namespace={namespace} />;
};

export default EnhancedArchivedExperiments;
