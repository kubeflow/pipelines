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
import AddIcon from '@material-ui/icons/Add';
import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import Paper from '@material-ui/core/Paper';
import PopOutIcon from '@material-ui/icons/Launch';
import RecurringRunsManager from './RecurringRunsManager';
import RunList from '../pages/RunList';
import Toolbar, { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import Tooltip from '@material-ui/core/Tooltip';
import { ApiExperiment } from '../apis/experiment';
import { ApiResourceType } from '../apis/job';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';

const css = stylesheet({
  card: {
    border: '1px solid #ddd',
    borderRadius: 8,
    height: 70,
    marginRight: 24,
    padding: '12px 20px 13px 24px',
  },
  cardActive: {
    border: `1px solid ${color.success}`,
  },
  cardBtn: {
    $nest: {
      '&:hover': {
        backgroundColor: 'initial',
      },
    },
    color: color.theme,
    fontSize: 12,
    minHeight: 16,
    paddingLeft: 0,
  },
  cardContent: {
    color: color.secondaryText,
    fontSize: 16,
    lineHeight: '28px',
  },
  cardRow: {
    borderBottom: '1px solid #eee',
    display: 'flex',
    flexFlow: 'row',
    minHeight: 120,
  },
  cardTitle: {
    color: color.strong,
    display: 'flex',
    fontSize: 12,
    fontWeight: 'bold',
    height: 20,
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  popOutIcon: {
    color: color.secondaryText,
    margin: -2,
    minWidth: 0,
    padding: 3,
  },
  recurringRunsActive: {
    color: '#0d652d',
  },
  recurringRunsCard: {
    width: 158,
  },
  recurringRunsDialog: {
    minWidth: 600,
  },
  runStatsCard: {
    width: 270,
  },
});

interface ExperimentDetailsState {
  activeRecurringRunsCount: number;
  experiment: ApiExperiment | null;
  recurringRunsManagerOpen: boolean;
  selectedRunIds: string[];
  selectedTab: number;
  runListToolbarProps: ToolbarProps;
}

class ExperimentDetails extends Page<{}, ExperimentDetailsState> {

  private _runlistRef = React.createRef<RunList>();

  private _runListToolbarActions: ToolbarActionConfig[] = [{
    action: () => this._startNewRun(false),
    icon: AddIcon,
    id: 'startNewRunBtn',
    outlined: true,
    title: 'Start new run',
    tooltip: 'Start a new run within this experiment',
  }, {
    action: () => this._startNewRun(true),
    icon: AddIcon,
    id: 'startNewRecurringRunBtn',
    outlined: true,
    title: 'Start recurring run',
    tooltip: 'Start a new recurring run in this experiment',
  }, {
    action: this._compareRuns.bind(this),
    disabled: true,
    disabledTitle: 'Select multiple runs to compare',
    id: 'compareBtn',
    title: 'Compare runs',
    tooltip: 'Compare up to 10 selected runs',
  }, {
    action: this._cloneRun.bind(this),
    disabled: true,
    disabledTitle: 'Select a run to clone',
    id: 'cloneBtn',
    title: 'Clone',
    tooltip: 'Create a copy from this run\s initial state',
  }];

  constructor(props: any) {
    super(props);

    this.state = {
      activeRecurringRunsCount: 0,
      experiment: null,
      recurringRunsManagerOpen: false,
      runListToolbarProps: {
        actions: this._runListToolbarActions,
        breadcrumbs: [{ displayName: 'Runs', href: '' }],
        topLevelToolbar: false,
      },
      // TODO: remove
      selectedRunIds: [],
      selectedTab: 0,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: this.refresh.bind(this),
        id: 'refreshBtn',
        title: 'Refresh',
        tooltip: 'Refresh',
      }],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: this.props.match.params[RouteParams.experimentId], href: '' }
      ],
    };
  }

  public render(): JSX.Element {
    const { activeRecurringRunsCount, experiment } = this.state;
    const description = experiment ? experiment.description || '' : '';

    return (
      <div className={classes(commonCss.page, padding(20, 'lrt'))}>

        {experiment && (
          <div className={commonCss.page}>
            <div className={css.cardRow}>
              <Paper id='recurringRunsCard' className={classes(
                css.card,
                css.recurringRunsCard,
                !!activeRecurringRunsCount && css.cardActive
              )} elevation={0}>
                <div>
                  <div className={css.cardTitle}>Recurring run configs</div>
                  <div className={classes(css.cardContent, !!activeRecurringRunsCount && css.recurringRunsActive)}>
                    {activeRecurringRunsCount + ' active'}
                  </div>
                  <Button className={css.cardBtn} disableRipple={true}
                    onClick={() => this.setState({ recurringRunsManagerOpen: true })}>
                    Manage
                    </Button>
                </div>
              </Paper>
              <Paper id='experimentDescriptionCard' className={classes(css.card, css.runStatsCard)} elevation={0}>
                <div className={css.cardTitle}>
                  <span>Experiment description</span>
                  <Button onClick={() => this.props.updateDialog({ title: 'Experiment description', content: description })}
                    className={classes(css.popOutIcon, 'popOutButton')}>
                    <Tooltip title='Read more'>
                      <PopOutIcon style={{ fontSize: 18 }} />
                    </Tooltip>
                  </Button>
                </div>
                {description.split('\n').slice(0, 2).map((line, i) =>
                  <div key={i} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {line}
                  </div>)
                }
                {description.split('\n').length > 2 ? '...' : ''}
              </Paper>
            </div>
            <Toolbar {...this.state.runListToolbarProps} />
            <RunList onError={this.showPageError.bind(this)}
              experimentIdMask={experiment.id} ref={this._runlistRef}
              selectedIds={this.state.selectedRunIds}
              onSelectionChange={this._selectionChanged.bind(this)} {...this.props} />

            <Dialog open={this.state.recurringRunsManagerOpen} classes={{ paper: css.recurringRunsDialog }}
              onClose={this._recurringRunsManagerClosed.bind(this)}>
              <DialogContent>
                <RecurringRunsManager {...this.props}
                  experimentId={this.props.match.params[RouteParams.experimentId]} />
              </DialogContent>
              <DialogActions>
                <Button onClick={this._recurringRunsManagerClosed.bind(this)} color='secondary'>
                  Close
                </Button>
              </DialogActions>
            </Dialog>
          </div>
        )}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();

    const experimentId = this.props.match.params[RouteParams.experimentId];

    try {
      const experiment = await Apis.experimentServiceApi.getExperiment(experimentId);

      this.props.updateToolbar({
        actions: this.props.toolbarProps.actions,
        breadcrumbs: [
          { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
          {
            displayName: experiment && experiment.name ? experiment.name : this.props.match.params[RouteParams.experimentId],
            href: ''
          }],
      });

      // TODO: get ALL jobs in the experiment
      const recurringRuns = await Apis.jobServiceApi.listJobs(
        undefined,
        100,
        '',
        ApiResourceType.EXPERIMENT.toString(),
        experimentId,
      );
      const activeRecurringRunsCount =
        (recurringRuns.jobs || []).filter(j => j.enabled === true).length;
      this.setState(
        { activeRecurringRunsCount, experiment },
        () => this._runlistRef.current && this._runlistRef.current.refresh()
      );
    } catch (err) {
      await this.showPageError(`Error: failed to retrieve experiment: ${experimentId}.`, err);
      logger.error(`Error loading experiment: ${experimentId}`, err);
    }
  }

  private _compareRuns(): void {
    const indices = this.state.selectedRunIds;
    if (indices.length > 1 && indices.length <= 10) {
      const runIds = this.state.selectedRunIds.join(',');
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.runlist]: runIds,
      });
      this.props.history.push(RoutePage.COMPARE + searchString);
    }
  }

  private _startNewRun(isRecurring: boolean): void {
    const searchString = new URLParser(this.props).build(Object.assign(
      { [QUERY_PARAMS.experimentId]: this.state.experiment!.id || '', },
      isRecurring ? { [QUERY_PARAMS.isRecurring]: '1' } : {}));
    this.props.history.push(RoutePage.NEW_RUN + searchString);
  }

  private _cloneRun(): void {
    if (this.state.selectedRunIds.length === 1) {
      const runId = this.state.selectedRunIds[0];
      const searchString = new URLParser(this.props).build({
        [QUERY_PARAMS.cloneFromRun]: runId || ''
      });
      this.props.history.push(RoutePage.NEW_RUN + searchString);
    }
  }

  private _selectionChanged(selectedRunIds: string[]): void {
    const toolbarActions = [...this.state.runListToolbarProps.actions];
    // Compare runs button
    toolbarActions[2].disabled = selectedRunIds.length <= 1 || selectedRunIds.length > 10;
    // Clone run button
    toolbarActions[3].disabled = selectedRunIds.length !== 1;
    this.setState({
      runListToolbarProps: {
        actions: toolbarActions,
        breadcrumbs: this.state.runListToolbarProps.breadcrumbs,
        topLevelToolbar: this.state.runListToolbarProps.topLevelToolbar,
      },
      selectedRunIds
    });
  }

  private _recurringRunsManagerClosed(): void {
    this.setState({ recurringRunsManagerOpen: false });
    // Reload the details to get any updated recurring runs
    this.refresh();
  }
}

export default ExperimentDetails;
