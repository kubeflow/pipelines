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
import DetailsTable from '../components/DetailsTable';
import RunUtils from '../lib/RunUtils';
import { ApiExperiment } from '../apis/experiment';
import { ApiJob } from '../apis/job';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarActionConfig, ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { formatDateString, enabledDisplayString, logger, errorToMessage } from '../lib/Utils';
import { triggerDisplayString } from '../lib/TriggerUtils';

interface RecurringRunConfigState {
  run: ApiJob | null;
}

class RecurringRunConfig extends Page<{}, RecurringRunConfigState> {

  constructor(props: any) {
    super(props);

    this.state = {
      run: null,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: this.refresh.bind(this),
        id: 'refreshBtn',
        title: 'Refresh',
        tooltip: 'Refresh',
      }, {
        action: () => this._setEnabledState(true),
        disabled: true,
        disabledTitle: 'Run schedule already enabled',
        id: 'enableBtn',
        title: 'Enable',
        tooltip: 'Enable the run\'s trigger',
      }, {
        action: () => this._setEnabledState(false),
        disabled: true,
        disabledTitle: 'Run schedule already disabled',
        id: 'disableBtn',
        title: 'Disable',
        tooltip: 'Disable the run\'s trigger',
      }, {
        action: () => this.props.updateDialog({
          buttons: [
            { onClick: () => this._deleteDialogClosed(true), text: 'Delete' },
            { onClick: () => this._deleteDialogClosed(false), text: 'Cancel' },
          ],
          onClose: () => this._deleteDialogClosed(false),
          title: 'Delete this recurring run?',
        }),
        id: 'deleteBtn',
        title: 'Delete',
        tooltip: 'Delete this recurring run',
      }],
      breadcrumbs: [],
    };
  }

  public render(): JSX.Element {
    const { run } = this.state;
    let runDetails: string[][] = [];
    let inputParameters: string[][] = [];
    let triggerDetails: string[][] = [];
    if (run && run.pipeline_spec) {
      runDetails = [
        ['Description', run.description!],
        ['Created at', formatDateString(run.created_at)],
      ];
      inputParameters = (run.pipeline_spec.parameters || []).map(p => [p.name || '', p.value || '']);
      if (run.trigger) {
        triggerDetails = [
          ['Enabled', enabledDisplayString(run.trigger, run.enabled!)],
          ['Trigger', triggerDisplayString(run.trigger)],
        ];
        if (run.max_concurrency) {
          triggerDetails.push(['Max. concurrent runs', run.max_concurrency]);
        }
        if (run.trigger.cron_schedule && run.trigger.cron_schedule.start_time) {
          triggerDetails.push(['Start time',
            formatDateString(run.trigger.cron_schedule.start_time)]);
        } else if (run.trigger.periodic_schedule && run.trigger.periodic_schedule.start_time) {
          triggerDetails.push(['Start time',
            formatDateString(run.trigger.periodic_schedule.start_time)]);
        }
        if (run.trigger.cron_schedule && run.trigger.cron_schedule.end_time) {
          triggerDetails.push(['End time',
            formatDateString(run.trigger.cron_schedule.end_time)]);
        } else if (run.trigger.periodic_schedule && run.trigger.periodic_schedule.end_time) {
          triggerDetails.push(['End time',
            formatDateString(run.trigger.periodic_schedule.end_time)]);
        }
      }
    }

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>

        {run && (
          <div className={commonCss.page}>
            <div className={commonCss.header}>Recurring run details</div>
            <DetailsTable fields={runDetails} />

            {!!triggerDetails.length && (
              <React.Fragment>
                <div className={commonCss.header}>Run trigger</div>
                <DetailsTable fields={triggerDetails} />
              </React.Fragment>
            )}

            {!!inputParameters.length && (
              <React.Fragment>
                <div className={commonCss.header}>Run parameters</div>
                <DetailsTable fields={inputParameters} />
              </React.Fragment>
            )}
          </div>
        )}
      </div>
    );
  }

  public componentDidMount(): Promise<void> {
    return this.load();
  }

  public async refresh(): Promise<void> {
    await this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const runId = this.props.match.params[RouteParams.runId];

    try {
      const run = await Apis.jobServiceApi.getJob(runId);
      const relatedExperimentId = RunUtils.getFirstExperimentReferenceId(run);
      let experiment: ApiExperiment | undefined;
      if (relatedExperimentId) {
        experiment = await Apis.experimentServiceApi.getExperiment(relatedExperimentId);
      }
      const breadcrumbs: Array<{ displayName: string, href: string }> = [];
      if (experiment) {
        breadcrumbs.push(
          { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
          {
            displayName: experiment.name!,
            href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experiment.id!)
          });
      } else {
        breadcrumbs.push(
          { displayName: 'All runs', href: RoutePage.RUNS }
        );
      }
      breadcrumbs.push({
        displayName: run ? run.name! : runId,
        href: '',
      });

      const toolbarActions = [...this.props.toolbarProps.actions];
      toolbarActions[1].disabled = run.enabled === true;
      toolbarActions[2].disabled = run.enabled === false;

      this.props.updateToolbar({ actions: toolbarActions, breadcrumbs });

      this.setState({ run });
    } catch (err) {
      await this.showPageError(`Error: failed to retrieve recurring run: ${runId}.`, err);
      logger.error(`Error loading recurring run: ${runId}`, err);
    }
  }

  private async _setEnabledState(enabled: boolean): Promise<void> {
    if (this.state.run) {
      const toolbarActions = [...this.props.toolbarProps.actions];

      const buttonIndex = enabled ? 1 : 2;
      const id = this.state.run.id!;

      toolbarActions[buttonIndex].busy = true;
      this._updateToolbar(toolbarActions);
      try {
        await (enabled ? Apis.jobServiceApi.enableJob(id) : Apis.jobServiceApi.disableJob(id));
        this.refresh();
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.showErrorDialog(
          `Failed to ${enabled ? 'enable' : 'disable'} recurring schedule`, errorMessage);
      } finally {
        toolbarActions[buttonIndex].busy = false;
        this._updateToolbar(toolbarActions);
      }
    }
  }

  private _updateToolbar(actions: ToolbarActionConfig[]): void {
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions });
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      // TODO: Show spinner during wait.
      try {
        await Apis.jobServiceApi.deleteJob(this.state.run!.id!);
        const breadcrumbs = this.props.toolbarProps.breadcrumbs;
        const previousPage = breadcrumbs.length ?
          breadcrumbs[breadcrumbs.length - 1].href : RoutePage.EXPERIMENTS;
        this.props.history.push(previousPage);
        this.props.updateSnackbar({
          message: `Successfully deleted recurring run: ${this.state.run!.name}`,
          open: true,
        });
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.showErrorDialog('Failed to delete recurring run', errorMessage);
        logger.error('Deleting recurring run failed with error:', err);
      }
    }
  }
}

export default RecurringRunConfig;
