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

import * as Apis from '../lib/Apis';
import * as React from 'react';
import * as UrlParser from '../lib/UrlParser';
import { BannerProps } from '../components/Banner';
import BusyButton from '../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Input from '../atoms/Input';
import MenuItem from '@material-ui/core/MenuItem';
import TextField, { TextFieldProps } from '@material-ui/core/TextField';
import { ToolbarProps } from '../components/Toolbar';
import Trigger from '../components/Trigger';
import { RouteComponentProps } from 'react-router';
import { DialogProps, RoutePage } from '../components/Router';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { apiJob, apiTrigger } from '../../../frontend/src/api/job';
import { apiPipeline } from '../../../frontend/src/api/pipeline';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';
import { SnackbarProps } from '@material-ui/core/Snackbar';

interface NewJobProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
  updateToolbar: (toolbarProps: ToolbarProps) => void;
}

interface NewJobState {
  description: string;
  errorMessage: string;
  isDeploying: boolean;
  jobName: string;
  maxConcurrentJobs?: string;
  pipelineId: string;
  pipelines: apiPipeline[];
  trigger: apiTrigger | undefined;
}

const css = stylesheet({
  errorMessage: {
    color: 'red',
  },
});

class NewJob extends React.Component<NewJobProps, NewJobState> {

  private _jobNameRef = React.createRef<HTMLInputElement>();

  constructor(props: any) {
    super(props);

    this.state = {
      description: '',
      errorMessage: '',
      isDeploying: false,
      jobName: '',
      maxConcurrentJobs: undefined,
      pipelineId: '',
      pipelines: [],
      trigger: undefined,
    };
  }

  public componentWillMount() {
    this.props.updateToolbar({
      actions: [],
      breadcrumbs: [
        { displayName: 'Jobs', href: RoutePage.JOBS },
        { displayName: 'New job', href: RoutePage.NEW_JOB }
      ]
    });
  }

  public componentDidMount() {
    this._load();
  }

  public componentWillUnmount() {
    this.props.updateBanner({});
  }

  public render() {
    const { pipelines, pipelineId, errorMessage } = this.state;
    const selectedPipeline = pipelines.find(p => p.id === pipelineId);

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>

        <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
          <div className={commonCss.header}>Job details</div>

          <Input label='Pipeline' required={true} instance={this} field='pipelineId'
            select={true} className='pipelineDropdown'>
            {pipelines.map((pipeline, i) => (
              <MenuItem key={i} value={pipeline.id}>
                {pipeline.name}
              </MenuItem>
            ))}
          </Input>

          <Input label='Job name' inputRef={this._jobNameRef} required={true} instance={this}
            field='jobName' autoFocus={true} />
          <Input label='Description' instance={this} field='description' />

          <br />

          <div className={commonCss.header}>Job trigger</div>

          <Trigger onChange={(trigger, maxConcurrentJobs) => this.setState({
            maxConcurrentJobs,
            trigger,
          }, this._validate.bind(this))} />

          {selectedPipeline && selectedPipeline.parameters &&
            !!selectedPipeline.parameters.length && <div>
              <div className={commonCss.header}>Input parameters</div>
              {selectedPipeline && (selectedPipeline.parameters || []).map((param, i) =>
                <TextField key={i} variant='outlined' label={param.name} value={param.value || ''}
                  onChange={(ev) => this._handleParamChange(i, ev.target.value || '')}
                  style={{ height: 40, maxWidth: 600 }} className={commonCss.textField} />)}
            </div>}

          <br />
          <div className={commonCss.flex}>
            <BusyButton title='Deploy' disabled={!!errorMessage} busy={this.state.isDeploying}
              className={classes(commonCss.actionButton, commonCss.primaryButton)}
              onClick={this._deploy.bind(this)} />
            <Button onClick={() => this.props.history.push(RoutePage.JOBS)}>Cancel</Button>
            <div className={css.errorMessage}>{errorMessage}</div>
          </div>
        </div>
      </div>
    );
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setState({
      [name]: value,
    } as any, () => {
      // Set querystring if pipeline id has changed
      if (name === 'pipelineId') {
        UrlParser.from('search')
          .set(UrlParser.QUERY_PARAMS.pipelineId, (value || '').toString());

        // Clear other query params so as not to confuse the user
        UrlParser.from('search').clear(UrlParser.QUERY_PARAMS.cloneFromJob);
      }

      this._validate();
    });
  }

  private _deploy(): void {
    const pipeline = this.state.pipelines.find(p => p.id === this.state.pipelineId);
    if (!pipeline) {
      this._showErrorDialog('Job deployment failed', `Could not find a pipeline with the selected id: ${this.state.pipelineId}`);
      logger.error(`Could not find a pipeline with the selected id: ${this.state.pipelineId}`);
      return;
    }
    const newJob: apiJob = {
      description: this.state.description,
      enabled: true,
      max_concurrency: this.state.maxConcurrentJobs || '1',
      name: this.state.jobName,
      parameters: pipeline.parameters,
      pipeline_id: pipeline.id,
      trigger: this.state.trigger,
    };

    this.setState({ isDeploying: true }, async () => {
      try {
        await Apis.newJob(newJob);
        this.props.history.push(RoutePage.JOBS);
        this.props.updateSnackbar({
          message: `Successfully deployed new Job: ${newJob.name}`,
          open: true,
        });
      } catch (err) {
        this._showErrorDialog('Job deployment failed', err.message);
        logger.error('Error deploying job:', err);
        this.setState({ isDeploying: false });
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

  private _handleParamChange(index: number, value: string) {
    const pipelines = this.state.pipelines;
    const pipeline = pipelines.find(p => p.id === this.state.pipelineId);
    if (!pipeline || !pipeline.parameters) {
      return;
    }
    pipeline.parameters[index].value = value;
    // TODO: is this doing anything?
    this.setState({ pipelines });
  }

  private _getCloneName(oldName: string) {
    const numberRegex = /Clone(?: \(([0-9]*)\))? of (.*)/;
    const match = oldName.match(numberRegex);
    if (!match) { // No match, add Clone prefix
      return 'Clone of ' + oldName;
    } else {
      const cloneNumber = match[1] ? +match[1] : 1;
      return `Clone (${cloneNumber + 1}) of ${match[2]}`;
    }
  }

  private async _load() {
    let response;
    try {
      response = await Apis.listPipelines({ pageSize: 25 });
    } catch (err) {
      this.props.updateBanner({
        additionalInfo: err.message,
        message: 'Error: failed to fetch pipelines. Click Details for more information.',
        mode: 'error',
        refresh: this._load.bind(this),
      });
      logger.error(`Cannot get the original job's data`);
      // Nothing else to do here if we couldn't load any pipelines
      return;
    }
    const pipelines = response.pipelines || [];
    this.setState({ pipelines });

    // Get clone job id from querystring if any
    const originalJobId = UrlParser.from('search').get(UrlParser.QUERY_PARAMS.cloneFromJob);
    const originalRunId = UrlParser.from('search').get(UrlParser.QUERY_PARAMS.cloneFromRun);
    if (originalJobId) {
      try {
        const job = await Apis.getJob(originalJobId);
        const pipelineIndex = pipelines.findIndex(p => p.id === job.pipeline_id);
        if (pipelineIndex === -1) {
          this.props.updateBanner({
            message: `Error: failed to find a pipeline corresponding to that of the original job: ${originalJobId}.`,
            mode: 'error',
          });
          logger.error(`Cannot get the original job's data`);
          return;
        }
        pipelines[pipelineIndex].parameters = job.parameters;
        this.setState({
          jobName: this._getCloneName(job.name!),
          pipelineId: job.pipeline_id!,
          pipelines,
        }, () => this._jobNameRef.current!.select());
        UrlParser.from('search').clear(UrlParser.QUERY_PARAMS.pipelineId);
      } catch (err) {
        this.props.updateBanner({
          additionalInfo: err.message,
          message: `Error: failed to get original job: ${originalJobId}. Click Details for more information.`,
          mode: 'error',
        });
        logger.error(`Failed to get original job ${originalJobId}`, err);
      }
      // Get the original run's id from querystring if any, get its job, and copy pipeline id and
      // parameters
    } else if (originalRunId) {
      try {
        const runDetails = await Apis.getRun(originalRunId);
        const job = await Apis.getJob(runDetails.run!.job_id!);
        const pipelineIndex = pipelines.findIndex(p => p.id === job.pipeline_id);
        if (pipelineIndex === -1) {
          this.props.updateBanner({
            message: `Error: failed to find a pipeline corresponding to that of the original run: ${originalRunId}.`,
            mode: 'error',
          });
          logger.error(`Cannot get the original run's data`);
          return;
        }

        const workflow = JSON.parse(runDetails.workflow || '{}') as Workflow;
        if (workflow.spec.arguments) {
          pipelines[pipelineIndex].parameters = workflow.spec.arguments.parameters;
        }

        this.setState({
          jobName: this._getCloneName(job.name!),
          pipelineId: job.pipeline_id!,
          pipelines,
        }, () => this._jobNameRef.current!.select());
      } catch (err) {
        this.props.updateBanner({
          additionalInfo: err.message,
          message: `Error: failed to get original run: ${originalRunId}. Click Details for more information.`,
          mode: 'error',
        });
        logger.error(`Failed to get original run ${originalRunId}`, err);
      }
    } else {
      // Get pipeline id from querystring if any
      let possiblePipelineId = UrlParser.from('search').get(UrlParser.QUERY_PARAMS.pipelineId);
      if (!pipelines.find(p => p.id === possiblePipelineId)) {
        possiblePipelineId = '';
        UrlParser.from('search').clear(UrlParser.QUERY_PARAMS.pipelineId);
      }
      this.setState({
        pipelineId: possiblePipelineId,
      });
    }

    this._validate();
  }

  private _validate() {
    // Validate state
    const { pipelineId, jobName, maxConcurrentJobs, trigger } = this.state;
    try {
      if (!pipelineId) {
        throw new Error('A pipeline must be selected');
      }
      if (!jobName) {
        throw new Error('Job name is required');
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

      if (hasTrigger && maxConcurrentJobs !== undefined &&
        !validMaxConcurrentJobs(maxConcurrentJobs)) {
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

export default NewJob;
