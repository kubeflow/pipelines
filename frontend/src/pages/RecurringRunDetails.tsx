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
import Buttons from '../lib/Buttons';
import DetailsTable from '../components/DetailsTable';
import RunUtils from '../lib/RunUtils';
import { ApiExperiment } from '../apis/experiment';
import { ApiJob } from '../apis/job';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { Breadcrumb, ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { formatDateString, enabledDisplayString, errorToMessage } from '../lib/Utils';
import { triggerDisplayString } from '../lib/TriggerUtils';

interface RecurringRunConfigState {
  run: ApiJob | null;
}

class RecurringRunDetails extends Page<{}, RecurringRunConfigState> {

  constructor(props: any) {
    super(props);

    this.state = {
      run: null,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: [
        buttons.refresh(this.refresh.bind(this)),
        buttons.enableRecurringRun(() => this.state.run ? this.state.run.id! : ''),
        buttons.disableRecurringRun(() => this.state.run ? this.state.run.id! : ''),
        buttons.delete(
          () => this.state.run ? [this.state.run!.id!] : [],
          'recurring run config',
          this._deleteCallback.bind(this),
          true, /* useCurrentResource */
        ),
      ],
      breadcrumbs: [],
      pageTitle: '',
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
            <DetailsTable title='Recurring run details' fields={runDetails} />

            {!!triggerDetails.length && (
              <DetailsTable title='Run trigger' fields={triggerDetails} />
            )}

            {!!inputParameters.length && (
              <DetailsTable title='Run parameters' fields={inputParameters} />
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

    let run: ApiJob;
    try {
      run = await Apis.jobServiceApi.getJob(runId);
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      await this.showPageError(
        `Error: failed to retrieve recurring run: ${runId}.`, new Error(errorMessage));
      return;
    }

    const relatedExperimentId = RunUtils.getFirstExperimentReferenceId(run);
    let experiment: ApiExperiment | undefined;
    if (relatedExperimentId) {
      try {
        experiment = await Apis.experimentServiceApi.getExperiment(relatedExperimentId);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showPageError(
          `Error: failed to retrieve this recurring run\'s experiment.`,
          new Error(errorMessage),
          'warning'
        );
      }
    }
    const breadcrumbs: Breadcrumb[] = [];
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
    const pageTitle = run ? run.name! : runId;

    const toolbarActions = [...this.props.toolbarProps.actions];
    toolbarActions[1].disabled = !!run.enabled;
    toolbarActions[2].disabled = !run.enabled;

    this.props.updateToolbar({ actions: toolbarActions, breadcrumbs, pageTitle });

    this.setState({ run });
  }

  private _deleteCallback(_: string[], success: boolean): void {
    if (success) {
      const breadcrumbs = this.props.toolbarProps.breadcrumbs;
      const previousPage = breadcrumbs.length ?
        breadcrumbs[breadcrumbs.length - 1].href : RoutePage.EXPERIMENTS;
      this.props.history.push(previousPage);
    }
  }
}

export default RecurringRunDetails;
