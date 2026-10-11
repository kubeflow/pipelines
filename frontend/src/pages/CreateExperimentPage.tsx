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
import { TextField } from 'src/components/ui/text-field';
import { Button } from 'src/components/ui/button';
import 'src/components/runs/ExperimentWorkflows.css';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { Apis } from 'src/lib/Apis';
import { Page, PageProps } from 'src/pages/Page';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import { URLParser } from 'src/lib/URLParser';
import { logger, errorToMessage } from 'src/lib/Utils';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { getLatestVersion } from 'src/pages/CreateRunForm';
import { CreateExperimentQueryPage } from 'src/pages/query/CreateExperimentQueryPage';
import { FeatureKey, isFeatureEnabled } from 'src/features';

interface CreateExperimentControllerState {
  description: string;
  validationError: string;
  isbeingCreated: boolean;
  experimentName: string;
  pipelineId?: string;
}

export class CreateExperimentController extends Page<
  { namespace?: string },
  CreateExperimentControllerState
> {
  private _experimentNameRef = React.createRef<HTMLInputElement>();

  constructor(props: any) {
    super(props);

    this.state = {
      description: '',
      experimentName: '',
      isbeingCreated: false,
      validationError: '',
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'New experiment',
    };
  }

  public render(): React.JSX.Element {
    const { description, experimentName, isbeingCreated, validationError } = this.state;

    return (
      <div className='kfp-workflow-page'>
        <div className='kfp-workflow-form'>
          <h2>Experiment details</h2>
          {/* TODO: this description needs work. */}
          <div className='kfp-workflow-description'>
            Think of an Experiment as a space that contains the history of all pipelines and their
            associated runs
          </div>

          <TextField
            id='experimentName'
            label='Experiment name'
            ref={this._experimentNameRef}
            required={true}
            onChange={this.handleChange('experimentName')}
            error={validationError}
            value={experimentName}
            autoFocus={true}
          />
          <TextField
            id='experimentDescription'
            label='Description'
            multiline={true}
            onChange={this.handleChange('description')}
            required={false}
            value={description}
          />

          <div className='kfp-workflow-actions'>
            <Button
              id='createExperimentBtn'
              disabled={!!validationError || isbeingCreated}
              aria-busy={isbeingCreated}
              onClick={this._create.bind(this)}
            >
              Next
            </Button>
            {isbeingCreated && <span role='status'>Creating experiment…</span>}
            <Button
              id='cancelNewExperimentBtn'
              variant='secondary'
              onClick={() => this.props.navigate(RoutePage.EXPERIMENTS)}
            >
              Cancel
            </Button>
          </div>
        </div>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }

  public async componentDidMount(): Promise<void> {
    this._isMounted = true;
    const urlParser = new URLParser(this.props);
    const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
    if (pipelineId) {
      this.setStateSafe({ pipelineId });
    }

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as HTMLInputElement | HTMLTextAreaElement).value;
    this.setState({ [name]: value } as any, this._validate.bind(this));
  };

  private _create(): void {
    const newExperiment: V2beta1Experiment = {
      description: this.state.description,
      display_name: this.state.experimentName,
      namespace: this.props.namespace,
    };

    this.setState({ isbeingCreated: true }, async () => {
      try {
        const response = await Apis.experimentServiceApiV2.createExperiment(newExperiment);
        let searchString = '';
        if (this.state.pipelineId) {
          const latestVersion = await getLatestVersion(this.state.pipelineId);
          searchString = new URLParser(this.props).build({
            [QUERY_PARAMS.experimentId]: response.experiment_id || '',
            [QUERY_PARAMS.pipelineId]: this.state.pipelineId,
            [QUERY_PARAMS.pipelineVersionId]: latestVersion?.pipeline_version_id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
        } else {
          searchString = new URLParser(this.props).build({
            [QUERY_PARAMS.experimentId]: response.experiment_id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
        }
        this.props.navigate(RoutePage.NEW_RUN + searchString);
        this.props.updateSnackbar({
          autoHideDuration: 10000,
          message: `Successfully created new Experiment: ${newExperiment.display_name}`,
          open: true,
        });
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showErrorDialog('Experiment creation failed', errorMessage);
        logger.error('Error creating experiment:', err);
        this.setState({ isbeingCreated: false });
      }
    });
  }

  private _validate(): void {
    // Validate state
    const { experimentName } = this.state;
    try {
      if (!experimentName) {
        throw new Error('Experiment name is required');
      }
      this.setState({ validationError: '' });
    } catch (err) {
      this.setState({
        validationError: err instanceof Error ? err.message : 'Experiment name is required',
      });
    }
  }
}

const CreateExperimentPage: React.FC<PageProps> = (props) => {
  const namespace = React.useContext(NamespaceContext);
  return isFeatureEnabled(FeatureKey.FUNCTIONAL_COMPONENT) ? (
    <CreateExperimentQueryPage {...props} namespace={namespace} />
  ) : (
    <CreateExperimentController {...props} namespace={namespace} />
  );
};

export default CreateExperimentPage;
