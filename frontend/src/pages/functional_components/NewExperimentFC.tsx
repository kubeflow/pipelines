/*
 * Copyright 2023 The Kubeflow Authors
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

import Button from '@material-ui/core/Button';
import React, { useEffect, useState } from 'react';
import { useMutation, useQuery } from 'react-query';
import { commonCss, fontsize, padding } from 'src/Css';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import BusyButton from 'src/atoms/BusyButton';
import Input from 'src/atoms/Input';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage } from 'src/lib/Utils';
import { getLatestVersion } from 'src/pages/NewRunV2';
import { PageProps } from 'src/pages/Page';
import { classes, stylesheet } from 'typestyle';

const css = stylesheet({
  errorMessage: {
    color: 'red',
  },
  // TODO: move to Css.tsx and probably rename.
  explanation: {
    fontSize: fontsize.small,
  },
});

interface ExperimentProps {
  namespace?: string;
}

type NewExperimentFCProps = ExperimentProps & PageProps;

export function NewExperimentFC(props: NewExperimentFCProps) {
  const urlParser = new URLParser(props);
  const { namespace, updateDialog, updateSnackbar, updateToolbar } = props;
  const [description, setDescription] = useState<string>('');
  const [experimentName, setExperimentName] = useState<string>('');
  const [isbeingCreated, setIsBeingCreated] = useState<boolean>(false);
  const [experimentResponse, setExperimentResponse] = useState<V2beta1Experiment>();
  const [errMsgFromApi, setErrMsgFromApi] = useState<string>();
  const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);

  const { data: latestVersion } = useQuery<V2beta1PipelineVersion | undefined, Error>(
    ['pipeline_versions', pipelineId],
    () => getLatestVersion(pipelineId!),
    { enabled: !!pipelineId },
  );

  useEffect(() => {
    updateToolbar({
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'New experiment',
    });
    // Initialize toolbar only once during the first render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Handle the redirection work when createExperiment is succeed
  useEffect(() => {
    if (experimentResponse) {
      const searchString = pipelineId
        ? new URLParser(props).build({
            [QUERY_PARAMS.experimentId]: experimentResponse.experiment_id || '',
            [QUERY_PARAMS.pipelineId]: pipelineId,
            [QUERY_PARAMS.pipelineVersionId]: latestVersion?.pipeline_version_id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          })
        : new URLParser(props).build({
            [QUERY_PARAMS.experimentId]: experimentResponse.experiment_id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
      props.history.push(RoutePage.NEW_RUN + searchString);

      updateSnackbar({
        autoHideDuration: 10000,
        message: `Successfully created new Experiment: ${experimentResponse.display_name}`,
        open: true,
      });
    }
    // Only trigger this effect when search string parameters change.
    // Do not rerun this effect if updateSnackbar callback has changes to avoid re-rendering.
    // Do not rerun this effect if pipelineId has changes to avoid re-rendering.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [experimentResponse, latestVersion]);

  useEffect(() => {
    if (errMsgFromApi) {
      updateDialog({
        buttons: [{ text: 'Dismiss' }],
        onClose: () => setIsBeingCreated(false),
        content: errMsgFromApi,
        title: 'Experiment creation failed',
      });
    }
    // Do not rerun this effect if updateDialog callback has changes to avoid re-rendering.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [errMsgFromApi, updateDialog]);

  const newExperimentMutation = useMutation((experiment: V2beta1Experiment) => {
    return Apis.experimentServiceApiV2.createExperiment(experiment);
  });

  const createExperiment = () => {
    let newExperiment: V2beta1Experiment = {
      display_name: experimentName,
      description: description,
      namespace: namespace,
    };
    setIsBeingCreated(true);

    newExperimentMutation.mutate(newExperiment, {
      onSuccess: response => {
        setExperimentResponse(response);
        setErrMsgFromApi(undefined);
      },
      onError: async err => {
        setErrMsgFromApi(await errorToMessage(err));
      },
    });
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
        <div className={commonCss.header}>Experiment details</div>
        <div className={css.explanation}>
          Think of an Experiment as a space that contains the history of all pipelines and their
          associated runs
        </div>

        <Input
          id='experimentName'
          label='Experiment name'
          required={true}
          onChange={event => setExperimentName(event.target.value)}
          value={experimentName}
          autoFocus={true}
          variant='outlined'
        />
        <Input
          id='experimentDescription'
          label='Description'
          multiline={true}
          onChange={event => setDescription(event.target.value)}
          required={false}
          value={description}
          variant='outlined'
        />

        <div className={commonCss.flex}>
          <BusyButton
            id='createExperimentBtn'
            disabled={!experimentName}
            busy={isbeingCreated}
            className={commonCss.buttonAction}
            title={'Next'}
            onClick={createExperiment}
          />
          <Button
            id='cancelNewExperimentBtn'
            onClick={() => props.history.push(RoutePage.EXPERIMENTS)}
          >
            Cancel
          </Button>
          <div className={css.errorMessage}>
            {experimentName ? '' : 'Experiment name is required'}
          </div>
        </div>
      </div>
    </div>
  );
}
