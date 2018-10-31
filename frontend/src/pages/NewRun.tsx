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
import BusyButton from '../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Input from '../atoms/Input';
import MenuItem from '@material-ui/core/MenuItem';
import RunUtils from '../lib/RunUtils';
import TextField, { TextFieldProps } from '@material-ui/core/TextField';
import Trigger from '../components/Trigger';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline } from '../apis/pipeline';
import { ApiRun, ApiResourceReference, ApiRelationship, ApiResourceType } from '../apis/run';
import { ApiTrigger, ApiJob } from '../apis/job';
import { Apis, PipelineSortKeys } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';

interface NewRunState {
  description: string;
  errorMessage: string;
  experiment?: ApiExperiment;
  experimentName?: string;
  isBeingCreated: boolean;
  isFirstRunInExperiment: boolean;
  isRecurringRun: boolean;
  maxConcurrentRuns?: string;
  pipelineId: string;
  pipelines: ApiPipeline[];
  runName: string;
  trigger?: ApiTrigger;
}

class NewRun extends Page<{}, NewRunState> {

  constructor(props: any) {
    super(props);

    this.state = {
      description: '',
      errorMessage: '',
      isBeingCreated: false,
      isFirstRunInExperiment: false,
      isRecurringRun: false,
      pipelineId: '',
      pipelines: [],
      runName: '',
    };
  }

  public getInitialToolbarState() {
    return {
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'Start a new run', href: RoutePage.NEW_RUN }
      ]
    };
  }

  public render() {
    const {
      pipelines, pipelineId, errorMessage, experimentName, isRecurringRun, isFirstRunInExperiment
    } = this.state;
    const selectedPipeline = pipelines.find(p => p.id === pipelineId);

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={commonCss.scrollContainer}>

          <div className={commonCss.header}>Run details</div>

          <Input label='Pipeline' required={true} instance={this} field='pipelineId'
            select={true} SelectProps={{ classes: { select: 'pipelinesDropdown' } }}>
            {pipelines.map((pipeline, i) => (
              <MenuItem key={i} value={pipeline.id} className='pipelinesDropdownItem'>
                {pipeline.name}
              </MenuItem>
            ))}
          </Input>

          <Input label='Run name' required={true} instance={this} field='runName' autoFocus={true} />
          <Input label='Description (optional)' multiline={true} instance={this}
            field='description' height='auto' />

          {experimentName && (
            <div>
              <div>This run will be associated with the following experiment</div>
              <Input label='Experiment' instance={this} disabled={true} field='experimentName' />
            </div>
          )}

          {isRecurringRun && (
            <React.Fragment>
              <div className={commonCss.header}>Job trigger</div>

              <Trigger onChange={(trigger, maxConcurrentRuns) => this.setState({
                maxConcurrentRuns,
                trigger,
              }, this._validate.bind(this))} />
            </React.Fragment>
          )}

          <div className={commonCss.header}>Run parameters</div>
          <div>{this._runParametersMessage(selectedPipeline)}</div>

          {selectedPipeline && selectedPipeline.parameters && selectedPipeline.parameters.length && (
            <div>
              {selectedPipeline && (selectedPipeline.parameters || []).map((param, i) =>
                <TextField key={i} variant='outlined' label={param.name} value={param.value || ''}
                  onChange={(ev) => this._handleParamChange(i, ev.target.value || '')}
                  style={{ height: 40, maxWidth: 600 }} className={commonCss.textField} />)}
            </div>
          )}

          <div className={classes(commonCss.flex, padding(20, 'tb'))}>
            <BusyButton id='createBtn' disabled={!!errorMessage} busy={this.state.isBeingCreated}
              className={commonCss.buttonAction} title='Create'
              onClick={this._create.bind(this)} />
            <Button onClick={() => {
              this.props.history.push(
                !!this.state.experiment
                  ? RoutePage.EXPERIMENT_DETAILS.replace(
                      ':' + RouteParams.experimentId, this.state.experiment.id!)
                  : RoutePage.RUNS);}}>
              {isFirstRunInExperiment ? 'Skip this step' : 'Cancel'}
            </Button>
            <div style={{ color: 'red' }}>{errorMessage}</div>
          </div>
        </div>
      </div>
    );
  }

  public async load(): Promise<void> {
    let response;
    try {
      response = await Apis.pipelineServiceApi.listPipelines(
        undefined,
        100,
        PipelineSortKeys.CREATED_AT + ' desc',
      );
    } catch (err) {
      this._handlePageError('Error: failed to fetch pipelines.', err.message, this.load.bind(this));
      logger.error('Cannot get the original run\'s data');
      // Nothing else to do here if we couldn't load any pipelines
      return;
    }
    const pipelines = response.pipelines || [];
    const urlParser = new URLParser(this.props);
    let experimentId: string | null = urlParser.get(QUERY_PARAMS.experimentId);

    // Get clone run id from querystring if any
    const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
    if (originalRunId) {
      try {
        const runDetail = await Apis.runServiceApi.getRun(originalRunId);

        // If the querystring did not contain an experiment ID, try to get one from the run.
        if (!experimentId) {
          experimentId = RunUtils.getFirstExperimentReferenceId(runDetail.run);
        }

        const associatedPipelineId = RunUtils.getPipelineId(runDetail.run);
        if (runDetail.run && associatedPipelineId) {
          const pipelineIndex = pipelines.findIndex((p) => p.id === associatedPipelineId);
          if (pipelineIndex === -1) {
            this._handlePageError(
              'Error: failed to find a pipeline corresponding to that of the original run:'
              + ` ${originalRunId}.`);
            logger.error('Cannot get the original run\'s data');
            return;
          }
          const workflow = JSON.parse(runDetail.pipeline_runtime!.workflow_manifest || '{}') as Workflow;
          if (workflow.spec.arguments) {
            pipelines[pipelineIndex].parameters = workflow.spec.arguments.parameters;
          }
          this.setState({
            pipelineId: associatedPipelineId,
            runName: this._getCloneName(runDetail.run.name!),
          });
        }
      } catch (err) {
        this._handlePageError(`Error: failed to get original run: ${originalRunId}.`, err);
        logger.error(`Failed to get original run ${originalRunId}`, err);
      }
    } else {
      // Get pipeline id from querystring if any
      let possiblePipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
      if (!pipelines.find(p => p.id === possiblePipelineId)) {
        possiblePipelineId = '';
        urlParser.clear(QUERY_PARAMS.pipelineId);
      }
      this.setState({
        pipelineId: possiblePipelineId,
      });
    }

    let experiment: ApiExperiment | undefined;
    let experimentName: string | undefined;
    if (experimentId) {
      try {
        experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
        experimentName = experiment.name;

        const breadcrumbs = [
          { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
          {
            displayName: experimentName!,
            href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experimentId),
          },
          { displayName: 'Start a new run', href: RoutePage.NEW_RUN }
        ];
        this.props.updateToolbar({ actions: this.props.toolbarProps.actions, breadcrumbs });
      } catch (err) {
        this._handlePageError(`Error: failed to get associated experiment: ${experimentId}.`, err);
        logger.error(`Failed to get associated experiment ${experimentId}`, err);
      }
    }

    this.setState({
      experiment,
      experimentName,
      isFirstRunInExperiment: urlParser.get(QUERY_PARAMS.firstRunInExperiment) === '1',
      isRecurringRun: urlParser.get(QUERY_PARAMS.isRecurring) === '1',
      pipelines,
    });

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setState({
      [name]: value,
    } as any, () => {
      // Set querystring if pipeline id has changed
      if (name === 'pipelineId') {
        const urlParser = new URLParser(this.props);
        urlParser.set(QUERY_PARAMS.pipelineId, (value || '').toString());

        // Clear other query params so as not to confuse the user
        urlParser.clear(QUERY_PARAMS.cloneFromRun);
      }

      this._validate();
    });
  }

  private _runParametersMessage(selectedPipeline: ApiPipeline | undefined): string {
    if (selectedPipeline) {
      if (selectedPipeline.parameters && selectedPipeline.parameters.length) {
        return 'Specify parameters required by the pipeline';
      } else {
        return 'This pipeline has no parameters';
      }
    }
    return 'Parameters will appear after you select a pipeline';
  }

  private _create(): void {
    const pipeline = this.state.pipelines.find(p => p.id === this.state.pipelineId);
    if (!pipeline) {
      this._showErrorDialog('Run creation failed',
        `Could not find a pipeline with the selected id: ${this.state.pipelineId}`);
      logger.error(`Could not find a pipeline with the selected id: ${this.state.pipelineId}`);
      return;
    }
    const references: ApiResourceReference[] = [];
    if (this.state.experiment) {
      references.push({
        key: {
          id: this.state.experiment.id,
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      });
    }

    const isRecurringRun = this.state.isRecurringRun;
    const newRun: ApiRun | ApiJob = isRecurringRun ? {
      description: this.state.description,
      enabled: true,
      max_concurrency: this.state.maxConcurrentRuns || '1',
      name: this.state.runName,
      pipeline_spec: { parameters: pipeline.parameters, pipeline_id: pipeline.id },
      trigger: this.state.trigger,
    } : {
        description: this.state.description,
        name: this.state.runName,
        pipeline_spec: { parameters: pipeline.parameters, pipeline_id: pipeline.id },
        resource_references: references,
      };

    this.setState({ isBeingCreated: true }, async () => {
      try {
        await isRecurringRun ?
          Apis.jobServiceApi.createJob(newRun) :
          Apis.runServiceApi.createRun(newRun);
        if (this.state.experiment) {
          this.props.history.push(
            RoutePage.EXPERIMENT_DETAILS.replace(
              ':' + RouteParams.experimentId, this.state.experiment.id!));
        } else {
          this.props.history.push(RoutePage.RUNS);
        }
        this.props.updateSnackbar({
          message: `Successfully created new Run: ${newRun.name}`,
          open: true,
        });
      } catch (err) {
        this._showErrorDialog('Run creation failed', err.message);
        logger.error('Error creating Run:', err);
        this.setState({ isBeingCreated: false });
      }
    });
  }

  private _showErrorDialog(title: string, content: string): void {
    this.props.updateDialog({
      buttons: [{ text: 'Dismiss' }],
      content,
      title,
    });
  }

  private _handleParamChange(index: number, value: string): void {
    const pipelines = this.state.pipelines;
    const pipeline = pipelines.find(p => p.id === this.state.pipelineId);
    if (!pipeline || !pipeline.parameters) {
      return;
    }
    pipeline.parameters[index].value = value;
    // TODO: is this doing anything?
    this.setState({ pipelines });
  }

  private _getCloneName(oldName: string): string {
    const numberRegex = /Clone(?: \(([0-9]*)\))? of (.*)/;
    const match = oldName.match(numberRegex);
    if (!match) { // No match, add Clone prefix
      return 'Clone of ' + oldName;
    } else {
      const cloneNumber = match[1] ? +match[1] : 1;
      return `Clone (${cloneNumber + 1}) of ${match[2]}`;
    }
  }

  private _handlePageError(message: string, error?: Error, refreshFunc?: () => void): void {
    this.props.updateBanner({
      additionalInfo: error ? error.message : undefined,
      message: message + ((error && error.message) ? ' Click Details for more information.' : ''),
      mode: 'error',
      refresh: refreshFunc,
    });
  }

  private _validate(): void {
    // Validate state
    const { pipelineId, maxConcurrentRuns, runName, trigger } = this.state;
    try {
      if (!pipelineId) {
        throw new Error('A pipeline must be selected');
      }
      if (!runName) {
        throw new Error('Run name is required');
      }

      const hasTrigger = trigger && (!!trigger.cron_schedule || !!trigger.periodic_schedule);
      if (hasTrigger) {
        const startDate = !!trigger!.cron_schedule ?
          trigger!.cron_schedule!.start_time : trigger!.periodic_schedule!.start_time;
        const endDate = !!trigger!.cron_schedule ?
          trigger!.cron_schedule!.end_time : trigger!.periodic_schedule!.end_time;
        if (startDate && endDate && startDate > endDate) {
          throw new Error('End date/time cannot be earlier than start date/time');
        }
      }
      const validMaxConcurrentJobs = (input: string) =>
        !isNaN(Number.parseInt(input, 10)) && +input > 0;

      if (hasTrigger && maxConcurrentRuns !== undefined &&
        !validMaxConcurrentJobs(maxConcurrentRuns)) {
        throw new Error('For triggered jobs, maximum concurrent jobs must be a positive number');
      }

      this.setState({
        errorMessage: '',
      });
    } catch (err) {
      this.setState({
        errorMessage: err.message,
      });
    }
  }
}

export default NewRun;
