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
import FormControlLabel from '@material-ui/core/FormControlLabel';
import InputAdornment from '@material-ui/core/InputAdornment';
import Radio from '@material-ui/core/Radio';
import React, { useEffect, useRef, useState } from 'react';
import Dropzone from 'react-dropzone';
import { DocumentationCompilePipeline } from 'src/components/UploadPipelineDialog';
import { classes, stylesheet } from 'typestyle';
import BusyButton from 'src/atoms/BusyButton';
import Input from 'src/atoms/Input';
import { CustomRendererProps } from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { color, commonCss, padding, zIndex } from 'src/Css';
import { Apis, PipelineSortKeys, BuildInfo } from 'src/lib/Apis';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage } from 'src/lib/Utils';
import { PageProps } from 'src/pages/Page';
import PrivateSharedSelector from 'src/components/PrivateSharedSelector';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import PipelinesDialogV2 from 'src/components/PipelinesDialogV2';
import { useMutation, useQuery } from 'react-query';

interface NewPipelineVersionFCProps extends PageProps {
  buildInfo?: BuildInfo;
  namespace?: string;
}

type uploadingPipelineObj = {
  pipelineName: string;
  pipelineDescription: string;
  file: File;
  namespace?: string;
};

type uploadingVersionObj = {
  pipelineVersionName: string;
  pipelineId: string;
  file: File;
  pipelineVersionDescription: string;
};

export enum ImportMethod {
  LOCAL = 'local',
  URL = 'url',
}

const css = stylesheet({
  dropOverlay: {
    backgroundColor: color.lightGrey,
    border: '2px dashed #aaa',
    bottom: 0,
    left: 0,
    padding: '2.5em 0',
    position: 'absolute',
    right: 0,
    textAlign: 'center',
    top: 0,
    zIndex: zIndex.DROP_ZONE_OVERLAY,
  },
  errorMessage: {
    color: 'red',
  },
  nonEditableInput: {
    color: color.secondaryText,
  },
  selectorDialog: {
    // If screen is small, use calc(100% - 120px). If screen is big, use 1200px.
    maxWidth: 1200, // override default maxWidth to expand this dialog further
    minWidth: 680,
    width: 'calc(100% - 120px)',
  },
});

export function NewPipelineVersionFC(props: NewPipelineVersionFCProps) {
  const urlParser = new URLParser(props);
  const { buildInfo, namespace, updateDialog, updateToolbar } = props;
  const [isbeingCreated, setIsBeingCreated] = useState<boolean>();
  const [errorMessage, setErrorMessage] = useState<string>();
  const [pipelineDescription, setPipelineDescription] = useState<string>('');
  const [pipelineId, setPipelineId] = useState(urlParser.get(QUERY_PARAMS.pipelineId));
  const [pipelineName, setPipelineName] = useState<string>('');
  const [pipelineVersionName, setPipelineVersionName] = useState<string>('');
  const [pipelineVersionDescription, setPipelineVersionDescription] = useState<string>('');
  const [pipeline, setPipeline] = useState<V2beta1Pipeline>();
  const [codeSourceUrl, setCodeSourceUrl] = useState<string>('');
  const [importMethod, setImportMethod] = useState<ImportMethod>(ImportMethod.URL);
  const [fileName, setFileName] = useState<string>('');
  const [file, setFile] = useState<File | null>(null);
  const [packageUrl, setPackageUrl] = useState<string>();
  const [dropzoneActive, setDropzoneActive] = useState<boolean>();
  const [newPipeline, setNewPipeline] = useState<boolean>(!pipelineId);
  const [pipelineSelectorOpen, setPipelineSelectorOpen] = useState<boolean>();
  const [unconfirmedSelectedPipeline, setUnconfirmedSelectedPipeline] = useState<V2beta1Pipeline>();
  const [isPrivate, setIsPrivate] = useState<boolean>(!!buildInfo?.apiServerMultiUser);

  const [createPipelineResponse, setCreatePipelineResponse] = useState<V2beta1Pipeline>();
  const [pipelineVersionResponse, setPipelineVersionResponse] = useState<V2beta1PipelineVersion>();
  const [errMsgFromPipelineApi, setErrMsgFromPipelineApi] = useState<string>();
  const [errMsgFromVersionApi, setErrMsgFromVersionApi] = useState<string>();

  const pipelineSelectorColumns = [
    { label: 'Pipeline name', flex: 1, sortKey: PipelineSortKeys.NAME },
    { label: 'Description', flex: 2, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  const dropzoneRef = useRef<Dropzone & HTMLDivElement>(null);
  const pipelineVersionNameRef = React.createRef<HTMLInputElement>();
  const pipelineNameRef = React.createRef<HTMLInputElement>();
  const pipelineDescriptionRef = React.createRef<HTMLInputElement>();

  const { data: getPipelineResponse } = useQuery<V2beta1Pipeline>(
    ['existing_pipeline', pipelineId],
    () => {
      return Apis.pipelineServiceApiV2.getPipeline(pipelineId!);
    },
    { enabled: !!pipelineId },
  );

  useEffect(() => {
    updateToolbar({
      actions: {},
      breadcrumbs: [{ displayName: 'Pipeline Versions', href: RoutePage.NEW_PIPELINE_VERSION }],
      pageTitle: 'New Pipeline',
    });
    // Initialize toolbar only once during the first render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (getPipelineResponse) {
      setPipeline(getPipelineResponse);
    }
  }, [getPipelineResponse]);

  useEffect(() => {
    const currDate = new Date();
    if (pipeline) {
      setPipelineId(pipeline.pipeline_id || '');
      setPipelineName(pipeline.display_name || '');
      setPipelineVersionName(
        pipeline?.display_name + '_version_at_' + currDate.toISOString() || '',
      );
    }
    if (unconfirmedSelectedPipeline) {
      setPipelineId(unconfirmedSelectedPipeline.pipeline_id || '');
      setPipelineName(unconfirmedSelectedPipeline.display_name || '');
      setPipelineVersionName(
        unconfirmedSelectedPipeline.display_name + '_version_at_' + currDate.toISOString() || '',
      );
    }
    if (file) {
      setFileName(file.name);
      setPipelineName(pipelineName || file.name.split('.')[0]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pipeline, unconfirmedSelectedPipeline, file]);

  // When creating piepline is succeed.
  useEffect(() => {
    if (createPipelineResponse) {
      createPipelineVersion(createPipelineResponse.pipeline_id!);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [createPipelineResponse]);

  // When uploading/creating version is succeed.
  useEffect(() => {
    if (pipelineVersionResponse) {
      props.history.push(
        RoutePage.PIPELINE_DETAILS.replace(
          `:${RouteParams.pipelineId}`,
          pipelineVersionResponse.pipeline_id! /* pipeline id of this version */,
        ).replace(
          `:${RouteParams.pipelineVersionId}`,
          pipelineVersionResponse.pipeline_version_id!,
        ),
      );
      props.updateSnackbar({
        autoHideDuration: 10000,
        message: `Successfully created new pipeline version: ${pipelineVersionResponse.display_name}`,
        open: true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pipelineVersionResponse]);

  // When upload/create pipeline is failed.
  useEffect(() => {
    if (errMsgFromPipelineApi) {
      updateDialog({
        buttons: [{ text: 'Dismiss' }],
        onClose: () => setIsBeingCreated(false),
        content: errMsgFromPipelineApi,
        title: 'Pipeline creation failed',
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [errMsgFromPipelineApi]);

  // When upload/create/list version is failed.
  useEffect(() => {
    if (errMsgFromVersionApi) {
      updateDialog({
        buttons: [{ text: 'Dismiss' }],
        onClose: () => setIsBeingCreated(false),
        content: errMsgFromVersionApi,
        title: 'Pipeline version creation failed',
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [errMsgFromVersionApi]);

  useEffect(() => {
    if (newPipeline) {
      if (!packageUrl && !fileName) {
        setErrorMessage('Must specify either package url or file in .yaml, .zip, or .tar.gz');
        return;
      }
      if (!pipelineName) {
        setErrorMessage('Pipeline name is required');
        return;
      }
    } else {
      if (!pipeline) {
        setErrorMessage('Pipeline is required');
        return;
      }
      if (!pipelineVersionName) {
        setErrorMessage('Pipeline version name is required');
        return;
      }
      if (pipelineVersionName && pipelineVersionName.length > 100) {
        setErrorMessage('Pipeline version name must contain no more than 100 characters');
        return;
      }
      if (!packageUrl && !fileName) {
        setErrorMessage('Please specify either package url or file in .yaml, .zip, or .tar.gz');
        return;
      }
    }
    setErrorMessage('');
  }, [newPipeline, fileName, packageUrl, pipeline, pipelineName, pipelineVersionName]);

  const uploadPipelineMutation = useMutation((newUploadingPipelineObj: uploadingPipelineObj) => {
    const { pipelineName, pipelineDescription, file, namespace } = newUploadingPipelineObj;
    return Apis.uploadPipelineV2(pipelineName!, pipelineDescription, file!, namespace);
  });

  const createPipelineMutation = useMutation((newCreatingPipelineObj: V2beta1Pipeline) => {
    return Apis.pipelineServiceApiV2.createPipeline(newCreatingPipelineObj);
  });

  const uploadPipelineVersionMutation = useMutation(
    (newUploadingVersionObj: uploadingVersionObj) => {
      const {
        pipelineVersionName,
        pipelineId,
        file,
        pipelineVersionDescription,
      } = newUploadingVersionObj;
      return Apis.uploadPipelineVersionV2(
        pipelineVersionName!,
        pipelineId!,
        file!,
        pipelineVersionDescription,
      );
    },
  );

  const createPipelineVersionMutation = useMutation(
    (newCreatingVersionObj: V2beta1PipelineVersion) => {
      const { pipeline_id } = newCreatingVersionObj;
      return Apis.pipelineServiceApiV2.createPipelineVersion(pipeline_id!, newCreatingVersionObj);
    },
  );

  const createPipeline = () => {
    setIsBeingCreated(true);

    if (importMethod === ImportMethod.LOCAL) {
      const newUploadingPipelineObj: uploadingPipelineObj = {
        pipelineName: pipelineName || '',
        pipelineDescription,
        file: file!,
        namespace: buildInfo?.apiServerMultiUser && isPrivate ? namespace : undefined,
      };
      uploadPipelineMutation.mutate(newUploadingPipelineObj, {
        onSuccess: async response => {
          const listVersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
            response.pipeline_id!,
            undefined,
            1, // Only need the latest one
            'created_at desc',
          );
          if (
            listVersionsResponse.pipeline_versions &&
            listVersionsResponse.pipeline_versions.length > 0
          ) {
            setPipelineVersionResponse(listVersionsResponse.pipeline_versions[0]);
          } else {
            setErrMsgFromVersionApi('Pipeline is empty');
          }
        },
        onError: async err => {
          setErrMsgFromPipelineApi(await errorToMessage(err));
        },
      });
    } else {
      const newCreatingPipelineObj: V2beta1Pipeline = {
        description: pipelineDescription,
        display_name: pipelineName,
        namespace: buildInfo?.apiServerMultiUser && isPrivate ? namespace : undefined,
      };
      createPipelineMutation.mutate(newCreatingPipelineObj, {
        onSuccess: setCreatePipelineResponse,
        onError: async err => {
          setErrMsgFromPipelineApi(await errorToMessage(err));
        },
      });
    }
  };

  const createPipelineVersion = (id: string) => {
    if (importMethod === ImportMethod.LOCAL) {
      const uploadingVersionObj: uploadingVersionObj = {
        pipelineVersionName: pipelineVersionName || '',
        pipelineId: id,
        file: file!,
        pipelineVersionDescription: pipelineVersionDescription || '',
      };
      uploadPipelineVersionMutation.mutate(uploadingVersionObj, {
        onSuccess: setPipelineVersionResponse,
        onError: async err => {
          setErrMsgFromVersionApi(await errorToMessage(err));
        },
      });
    } else {
      const creatingVersionObj: V2beta1PipelineVersion = {
        pipeline_id: id,
        display_name: pipelineVersionName,
        description: pipelineVersionDescription,
        package_url: { pipeline_url: packageUrl },
      };
      createPipelineVersionMutation.mutate(creatingVersionObj, {
        onSuccess: setPipelineVersionResponse,
        onError: async err => {
          setErrMsgFromVersionApi(await errorToMessage(err));
        },
      });
    }
  };

  const pipelineSelectorClosed = (confirmed: boolean) => {
    if (confirmed && unconfirmedSelectedPipeline) {
      setPipeline(unconfirmedSelectedPipeline);
    }
    setPipelineSelectorOpen(false);
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
        <div className={classes(padding(10, 't'))}>Upload pipeline or pipeline version.</div>
        <div className={classes(commonCss.flex, padding(10, 'b'))}>
          <FormControlLabel
            id='createNewPipelineBtn'
            label='Create a new pipeline'
            checked={newPipeline === true}
            control={<Radio color='primary' />}
            onChange={() => {
              setCodeSourceUrl('');
              setNewPipeline(true);
              setPipelineDescription('');
              setPipelineName('');
              setPipelineVersionName('');
            }}
          />
          <FormControlLabel
            id='createPipelineVersionUnderExistingPipelineBtn'
            label='Create a new pipeline version under an existing pipeline'
            checked={newPipeline === false}
            control={<Radio color='primary' />}
            onChange={() => {
              setCodeSourceUrl('');
              setNewPipeline(false);
              setPipelineDescription('');
              setPipelineName('');
              setPipelineVersionName('');
            }}
          />
        </div>
        {newPipeline === true && buildInfo?.apiServerMultiUser && (
          <PrivateSharedSelector onChange={setIsPrivate}></PrivateSharedSelector>
        )}

        {/* Pipeline name and help text for uploading new pipeline */}
        {newPipeline === true && (
          <>
            <div>Upload pipeline with the specified package.</div>
            <Input
              id='newPipelineName'
              value={pipelineName}
              required={true}
              label='Pipeline Name'
              variant='outlined'
              inputRef={pipelineNameRef}
              onChange={event => setPipelineName(event.target.value)}
              autoFocus={true}
            />
            <Input
              id='pipelineDescription'
              value={pipelineDescription}
              required={false}
              label='Pipeline Description'
              variant='outlined'
              inputRef={pipelineDescriptionRef}
              onChange={event => setPipelineDescription(event.target.value)}
              autoFocus={true}
            />
            {/* Choose a local file for package or specify a url for package */}
          </>
        )}

        {/* Pipeline selector and help text for uploading new pipeline version */}
        {newPipeline === false && (
          <>
            <div>Upload pipeline version with the specified package.</div>
            {/* Select pipeline */}
            <Input
              value={pipelineName}
              required={true}
              label='Pipeline'
              disabled={true}
              variant='outlined'
              inputRef={pipelineNameRef}
              onChange={event => setPipelineName(event.target.value)}
              autoFocus={true}
              InputProps={{
                classes: { disabled: css.nonEditableInput },
                endAdornment: (
                  <InputAdornment position='end'>
                    <Button
                      color='secondary'
                      id='choosePipelineBtn'
                      onClick={() => setPipelineSelectorOpen(true)}
                      style={{ padding: '3px 5px', margin: 0 }}
                    >
                      Choose
                    </Button>
                  </InputAdornment>
                ),
                readOnly: true,
              }}
            />

            <PipelinesDialogV2
              {...props}
              open={!!pipelineSelectorOpen}
              selectorDialog={css.selectorDialog}
              onClose={(confirmed, selectedPipeline?: V2beta1Pipeline) => {
                setUnconfirmedSelectedPipeline(selectedPipeline);
                pipelineSelectorClosed(confirmed);
              }}
              namespace={namespace}
              pipelineSelectorColumns={pipelineSelectorColumns}
            ></PipelinesDialogV2>

            {/* Set pipeline version name */}
            <Input
              id='pipelineVersionName'
              label='Pipeline Version name'
              inputRef={pipelineVersionNameRef}
              required={true}
              onChange={event => setPipelineVersionName(event.target.value)}
              value={pipelineVersionName}
              autoFocus={true}
              variant='outlined'
            />
            <Input
              id='pipelineVersionDescription'
              value={pipelineVersionDescription}
              required={false}
              label='Pipeline Version Description'
              variant='outlined'
              onChange={event => setPipelineVersionDescription(event.target.value)}
              autoFocus={true}
            />
          </>
        )}

        {/* Different package explanation based on import method*/}
        {importMethod === ImportMethod.LOCAL && (
          <>
            <div className={padding(10, 'b')}>
              Choose a pipeline package file from your computer, and give the pipeline a unique
              name.
              <br />
              You can also drag and drop the file here.
            </div>
            <DocumentationCompilePipeline />
          </>
        )}
        {importMethod === ImportMethod.URL && (
          <>
            <div className={padding(10, 'b')}>URL must be publicly accessible.</div>
            <DocumentationCompilePipeline />
          </>
        )}

        {/* Different package input field based on import method*/}
        <div className={classes(commonCss.flex, padding(10, 'b'))}>
          <FormControlLabel
            id='localPackageBtn'
            label='Upload a file'
            checked={importMethod === ImportMethod.LOCAL}
            control={<Radio color='primary' />}
            onChange={() => setImportMethod(ImportMethod.LOCAL)}
          />
          <Dropzone
            id='dropZone'
            disableClick={true}
            onDrop={(files: File[]) => {
              setDropzoneActive(false);
              setFile(files[0]);
            }}
            onDragEnter={() => {
              setDropzoneActive(true);
            }}
            onDragLeave={() => {
              setDropzoneActive(false);
            }}
            style={{ position: 'relative' }}
            ref={dropzoneRef}
            inputProps={{ tabIndex: -1 }}
            disabled={importMethod === ImportMethod.URL}
          >
            {dropzoneActive && <div className={css.dropOverlay}>Drop files..</div>}
            <Input
              data-testid='uploadFileInput'
              onChange={event => setFileName(event.target.value)}
              value={fileName}
              required={true}
              label='File'
              variant='outlined'
              disabled={importMethod === ImportMethod.URL}
              // Find a better to align this input box with others
              InputProps={{
                endAdornment: (
                  <InputAdornment position='end'>
                    <Button
                      color='secondary'
                      onClick={() => dropzoneRef.current!.open()}
                      style={{ padding: '3px 5px', margin: 0, whiteSpace: 'nowrap' }}
                      disabled={importMethod === ImportMethod.URL}
                    >
                      Choose file
                    </Button>
                  </InputAdornment>
                ),
                readOnly: true,
                style: {
                  maxWidth: 2000,
                  width: 455,
                },
              }}
            />
          </Dropzone>
        </div>
        <div className={classes(commonCss.flex, padding(10, 'b'))}>
          <FormControlLabel
            id='remotePackageBtn'
            label='Import by url'
            checked={importMethod === ImportMethod.URL}
            control={<Radio color='primary' />}
            onChange={() => setImportMethod(ImportMethod.URL)}
          />
          <Input
            id='pipelinePackageUrl'
            label='Package Url'
            multiline={true}
            onChange={event => setPackageUrl(event.target.value)}
            value={packageUrl}
            variant='outlined'
            disabled={importMethod === ImportMethod.LOCAL}
            // Find a better to align this input box with others
            style={{
              maxWidth: 2000,
              width: 465,
            }}
          />
        </div>

        {/* Fill pipeline version code source url */}
        <Input
          id='pipelineVersionCodeSource'
          label='Code Source'
          multiline={true}
          onChange={event => setCodeSourceUrl(event.target.value)}
          required={false}
          value={codeSourceUrl}
          variant='outlined'
        />

        {/* Create pipeline or pipeline version */}
        <div className={commonCss.flex}>
          <BusyButton
            id='createNewPipelineOrVersionBtn'
            disabled={!!errorMessage}
            busy={isbeingCreated}
            className={commonCss.buttonAction}
            title={'Create'}
            onClick={() => {
              newPipeline ? createPipeline() : createPipelineVersion(pipelineId!);
            }}
          />
          <Button
            id='cancelNewPipelineOrVersionBtn'
            onClick={() => props.history.push(RoutePage.PIPELINES)}
          >
            Cancel
          </Button>
          <div className={css.errorMessage}>{errorMessage}</div>
        </div>
      </div>
    </div>
  );
}

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = props => {
  return <Description description={props.value || ''} forceInline={true} />;
};
