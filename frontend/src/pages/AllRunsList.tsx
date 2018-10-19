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

import * as React from 'react';
import CloneIcon from '@material-ui/icons/FileCopy';
import CompareIcon from '@material-ui/icons/CompareArrows';
import RefreshIcon from '@material-ui/icons/Refresh';
import RunList from './RunList';
import { BannerProps } from '../components/Banner';
import { RoutePage } from '../components/Router';
import { RouteComponentProps } from 'react-router';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';

interface AllRunsListProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

interface AllRunsListState {
  selectedIds: string[];
}

class AllRunsList extends React.Component<AllRunsListProps, AllRunsListState> {

  private _runlistRef = React.createRef<RunList>();
  private _toolbarActions: ToolbarActionConfig[] = [
    {
      action: this._compareRuns.bind(this),
      disabled: true,
      icon: CompareIcon,
      id: 'compareBtn',
      title: 'Compare runs',
      tooltip: 'Compare up to 10 selected runs',
    },
    {
      action: this._refresh.bind(this),
      disabled: false,
      icon: RefreshIcon,
      id: 'refreshBtn',
      title: 'Refresh',
      tooltip: 'Refresh',
    },
    {
      action: this._cloneRun.bind(this),
      disabled: true,
      disabledTitle: 'Select a run to clone',
      icon: CloneIcon,
      id: 'cloneBtn',
      title: 'Clone',
      tooltip: 'Clone',
    },
  ];

  constructor(props: any) {
    super(props);

    this.state = {
      selectedIds: [],
    };
  }

  public componentWillMount() {
    this.props.updateToolbar({ actions: this._toolbarActions, breadcrumbs: [{ displayName: 'All runs', href: '' }] });
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render() {
    return <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <RunList handleError={this._handlePageError.bind(this)} selectedIds={this.state.selectedIds}
        updateSelection={this._selectionChanged.bind(this)}
        {...this.props} ref={this._runlistRef} />
    </div>;
  }

  private _compareRuns() {
    const indices = this.state.selectedIds;
    if (indices.length > 1 && indices.length <= 10) {
      const runIds = this.state.selectedIds.join(',');
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.runlist]: runIds,
      });
      this.props.history.push(RoutePage.COMPARE + searchString);
    }
  }

  private async _refresh() {
    // Tell run list to refresh
    if (this._runlistRef.current) {
      this._runlistRef.current.refresh();
    }
  }

  private _selectionChanged(selectedIds: string[]) {
    const toolbarActions = [...this.props.toolbarProps.actions];
    toolbarActions[0].disabled = selectedIds.length <= 1 || selectedIds.length > 10;
    toolbarActions[2].disabled = selectedIds.length !== 1;
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setState({ selectedIds });
  }

  private _handlePageError(message: string, error: Error): void {
    this.props.updateBanner({
      additionalInfo: error.message,
      message,
      mode: 'error',
      refresh: this._refresh.bind(this),
    });
  }

  private _cloneRun() {
    if (this.state.selectedIds.length === 1) {
      const runId = this.state.selectedIds[0];
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromRun]: runId || ''
      });
      this.props.history.push(RoutePage.NEW_JOB + searchString);
    }
  }
}

export default AllRunsList;
