/*
 * Copyright 2019 The Kubeflow Authors
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

import Button from '@mui/material/Button';
import FormControlLabel from '@mui/material/FormControlLabel';
import InputAdornment from '@mui/material/InputAdornment';
import Radio from '@mui/material/Radio';
import { TextFieldProps } from '@mui/material/TextField';
import * as React from 'react';
import Dropzone from 'react-dropzone';
import { DocumentationCompilePipeline } from 'src/components/UploadPipelineDialog';
import { classes, stylesheet } from 'typestyle';
import BusyButton from 'src/atoms/BusyButton';
import Input from 'src/atoms/Input';
import { CustomRendererProps } from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { color, commonCss, padding, zIndex } from 'src/Css';
import { Apis, PipelineSortKeys, BuildInfo } from 'src/lib/Apis';
import { useURLParser } from 'src/lib/URLParser';
import { ensureError, errorToMessage, logger } from 'src/lib/Utils';
import { PageProps } from './Page';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import PrivateSharedSelector from 'src/components/PrivateSharedSelector';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import PipelinesDialogV2 from 'src/components/PipelinesDialogV2';
import { NameWithTooltip } from 'src/components/CustomTableNameColumn';

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return <Description description={props.value || ''} forceInline={true} />;
};

export interface NewPipelineVersionProps extends PageProps {
  buildInfo?: BuildInfo;
  namespace?: string;
}

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

function getK8sNameRegex(): RegExp {
  return /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;
}

function NewPipelineVersion(props: NewPipelineVersionProps) {
  const urlParser = useURLParser();

  // Refs
  const pipelineVersionNameRef = React.useRef<HTMLInputElement>(null);
  const pipelineVersionDisplayNameRef = React.useRef<HTMLInputElement>(null);
  const pipelineNameRef = React.useRef<HTMLInputElement>(null);
  const pipelineDisplayNameRef = React.useRef<HTMLInputElement>(null);
  const pipelineDescriptionRef = React.useRef<HTMLInputElement>(null);

  // Initial setup
  const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
  const initialIsPrivate = props.buildInfo?.apiServerMultiUser ? true : false;

  // State hooks
  const [validationError, setValidationError] = React.useState<string>('');
  const [isbeingCreated, setIsBeingCreated] = React.useState<boolean>(false);
  const [pipelineDescription, setPipelineDescription] = React.useState<string>('');
  const [currentPipelineId, setCurrentPipelineId] = React.useState<string>('');
  const [pipelineName, setPipelineName] = React.useState<string>('');
  const [pipelineDisplayName, setPipelineDisplayName] = React.useState<string>('');
  const [pipelineVersionName, setPipelineVersionName] = React.useState<string>('');
  const [pipelineVersionDisplayName, setPipelineVersionDisplayName] = React.useState<string>('');
  const [pipelineVersionDescription, setPipelineVersionDescription] = React.useState<string>('');
  const [pipeline, setPipeline] = React.useState<V2beta1Pipeline | undefined>();
  const [codeSourceUrl, setCodeSourceUrl] = React.useState<string>('');
  const [importMethod, setImportMethod] = React.useState<ImportMethod>(ImportMethod.URL);
  const [fileName, setFileName] = React.useState<string>('');
  const [file, setFile] = React.useState<File | null>(null);
  const [packageUrl, setPackageUrl] = React.useState<string>('');
  const [dropzoneActive, setDropzoneActive] = React.useState<boolean>(false);
  const [newPipeline, setNewPipeline] = React.useState<boolean>(pipelineId ? false : true);
  const [pipelineSelectorOpen, setPipelineSelectorOpen] = React.useState<boolean>(false);
  const [unconfirmedSelectedPipeline, setUnconfirmedSelectedPipeline] = React.useState<
    V2beta1Pipeline | undefined
  >();
  const [isPrivate, setIsPrivate] = React.useState<boolean>(initialIsPrivate);

  // Constants
  const pipelineSelectorColumns = [
    {
      label: 'Pipeline name',
      flex: 1,
      sortKey: PipelineSortKeys.DISPLAY_NAME,
      customRenderer: NameWithTooltip,
    },
    { label: 'Description', flex: 2, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  // Utility functions
  const showErrorDialog = React.useCallback(
    async (title: string, content: string) => {
      props.updateDialog({
        buttons: [{ text: 'Dismiss' }],
        content,
        title,
      });
    },
    [props],
  );

  const validate = React.useCallback(() => {
    // Validate state
    // 3 valid use case for now:
    // (1) new pipeline (and a default version) from local file
    // (2) new pipeline (and a default version) from url
    // (3) new pipeline version (under an existing pipeline) from url
    try {
      if (newPipeline) {
        if (!packageUrl && !fileName) {
          throw new Error('Must specify either package url  or file in .yaml, .zip, or .tar.gz');
        }
        if (!pipelineName) {
          throw new Error('Pipeline name is required');
        }
        if (props.buildInfo?.pipelineStore === 'kubernetes') {
          if (!getK8sNameRegex().test(pipelineName)) {
            throw new Error(
              'Pipeline name must match Kubernetes naming pattern: lowercase letters, numbers, hyphens, and dots',
            );
          }
        }
      } else {
        if (!pipeline) {
          throw new Error('Pipeline is required');
        }
        if (!pipelineVersionName) {
          throw new Error('Pipeline version name name is required');
        }
        if (pipelineVersionName && pipelineVersionName.length > 100) {
          throw new Error('Pipeline version name must contain no more than 100 characters');
        }
        if (props.buildInfo?.pipelineStore === 'kubernetes') {
          if (!getK8sNameRegex().test(pipelineVersionName)) {
            throw new Error(
              'Pipeline version name must match Kubernetes naming pattern: lowercase letters, numbers, hyphens, and dots',
            );
          }
        }
        if (!packageUrl && !fileName) {
          throw new Error('Please specify either package url or file in .yaml, .zip, or .tar.gz');
        }
      }
      setValidationError('');
    } catch (err) {
      const error = ensureError(err);
      setValidationError(error.message);
    }
  }, [
    newPipeline,
    packageUrl,
    fileName,
    pipelineName,
    props.buildInfo,
    pipeline,
    pipelineVersionName,
  ]);

  const handleChange = React.useCallback(
    (name: string) => (event: any) => {
      const value = (event.target as TextFieldProps).value as string;

      // Update the corresponding state
      switch (name) {
        case 'pipelineDescription':
          setPipelineDescription(value);
          break;
        case 'pipelineName':
          setPipelineName(value);
          // When pipeline name is changed, suggest a version name based on pipeline name
          const currDate = new Date();
          setPipelineVersionName(
            value +
              '-version-at-' +
              currDate
                .toISOString()
                .toLowerCase()
                .replace(/:/g, '-'),
          );
          break;
        case 'pipelineDisplayName':
          setPipelineDisplayName(value);
          break;
        case 'pipelineVersionName':
          setPipelineVersionName(value);
          break;
        case 'pipelineVersionDisplayName':
          setPipelineVersionDisplayName(value);
          break;
        case 'pipelineVersionDescription':
          setPipelineVersionDescription(value);
          break;
        case 'packageUrl':
          setPackageUrl(value);
          break;
        case 'codeSourceUrl':
          setCodeSourceUrl(value);
          break;
        default:
          break;
      }
    },
    [],
  );

  // Effect to validate whenever relevant state changes
  React.useEffect(() => {
    validate();
  }, [validate]);

  const pipelineSelectorClosed = React.useCallback(
    async (confirmed: boolean) => {
      let selectedPipeline = pipeline;
      const currDate = new Date();
      if (confirmed && unconfirmedSelectedPipeline) {
        selectedPipeline = unconfirmedSelectedPipeline;
      }

      setPipeline(selectedPipeline);
      setCurrentPipelineId((selectedPipeline && selectedPipeline.pipeline_id) || '');
      setPipelineName((selectedPipeline && selectedPipeline.name) || '');
      setPipelineDisplayName((selectedPipeline && selectedPipeline.display_name) || '');
      setPipelineSelectorOpen(false);
      // Suggest a version name based on pipeline name
      setPipelineVersionName(
        (selectedPipeline &&
          selectedPipeline.name +
            '-version-at-' +
            currDate
              .toISOString()
              .toLowerCase()
              .replace(/:/g, '-')) ||
          '',
      );
      setUnconfirmedSelectedPipeline(undefined);
    },
    [pipeline, unconfirmedSelectedPipeline],
  );

  const onDrop = React.useCallback((files: File[]) => {
    setFile(files[0]);
    setFileName(files[0].name);
    setPackageUrl('');
    setImportMethod(ImportMethod.LOCAL);
  }, []);

  const onDropForTest = React.useCallback(
    (files: File[]) => {
      onDrop(files);
    },
    [onDrop],
  );

  const createPipelineVersion = React.useCallback(async (): Promise<V2beta1PipelineVersion> => {
    if (importMethod === ImportMethod.LOCAL) {
      if (!file) {
        throw new Error('File should be selected');
      }
      return Apis.uploadPipelineVersionV2(
        pipelineVersionName,
        pipelineVersionDisplayName,
        currentPipelineId!,
        file,
        pipelineVersionDescription,
      );
    } else {
      // importMethod === ImportMethod.URL
      let newPipelineVersion: V2beta1PipelineVersion = {
        pipeline_id: currentPipelineId,
        display_name: pipelineVersionDisplayName,
        name: pipelineVersionName,
        description: pipelineVersionDescription,
        package_url: { pipeline_url: packageUrl },
      };
      return Apis.pipelineServiceApiV2.createPipelineVersion(
        currentPipelineId!,
        newPipelineVersion,
      );
    }
  }, [
    importMethod,
    file,
    pipelineVersionName,
    pipelineVersionDisplayName,
    currentPipelineId,
    pipelineVersionDescription,
    packageUrl,
  ]);

  const create = React.useCallback(async () => {
    setIsBeingCreated(true);
    try {
      let namespace: undefined | string;
      if (props.buildInfo?.apiServerMultiUser) {
        if (isPrivate) {
          namespace = props.namespace;
        }
      }
      // 3 use case for now:
      // (1) new pipeline (and a default version) from local file
      // (2) new pipeline (and a default version) from url
      // (3) new pipeline version (under an existing pipeline) from url
      let pipelineVersionResponse: V2beta1PipelineVersion;
      if (newPipeline && importMethod === ImportMethod.LOCAL) {
        const pipelineResponse = await Apis.uploadPipelineV2(
          pipelineName!,
          pipelineDisplayName!,
          pipelineDescription,
          file!,
          namespace,
        );
        const listVersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
          pipelineResponse.pipeline_id!,
          undefined,
          1, // Only need the latest one
          'created_at desc',
        );
        if (listVersionsResponse.pipeline_versions) {
          pipelineVersionResponse = listVersionsResponse.pipeline_versions[0];
        } else {
          throw new Error('Pipeline is empty');
        }
      } else if (newPipeline && importMethod === ImportMethod.URL) {
        const newPipelineData: V2beta1Pipeline = {
          description: pipelineDescription,
          display_name: pipelineName,
          name: pipelineName,
          namespace,
        };
        const createPipelineResponse = await Apis.pipelineServiceApiV2.createPipeline(
          newPipelineData,
        );
        setCurrentPipelineId(createPipelineResponse.pipeline_id!);
        pipelineVersionResponse = await createPipelineVersion();
      } else {
        pipelineVersionResponse = await createPipelineVersion();
      }

      // If success, go to pipeline details page of the new version
      props.navigate(
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
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      await showErrorDialog('Pipeline version creation failed', errorMessage);
      logger.error('Error creating pipeline version:', err);
      setIsBeingCreated(false);
    }
  }, [
    props,
    isPrivate,
    newPipeline,
    importMethod,
    pipelineName,
    pipelineDisplayName,
    pipelineDescription,
    file,
    createPipelineVersion,
    showErrorDialog,
  ]);

  // Initialize toolbar
  React.useEffect(() => {
    const toolbarProps: ToolbarProps = {
      actions: {},
      breadcrumbs: [{ displayName: 'Pipeline Versions', href: RoutePage.NEW_PIPELINE_VERSION }],
      pageTitle: 'New Pipeline',
    };
    props.updateToolbar(toolbarProps);
  }, [props]);

  // Load pipeline data on mount if pipelineId is provided
  React.useEffect(() => {
    const loadPipelineData = async () => {
      const pipelineIdFromUrl = urlParser.get(QUERY_PARAMS.pipelineId);
      if (pipelineIdFromUrl) {
        try {
          const pipelineResponse = await Apis.pipelineServiceApiV2.getPipeline(pipelineIdFromUrl);
          setCurrentPipelineId(pipelineIdFromUrl);
          setPipelineName(pipelineResponse.display_name || '');
          setPipeline(pipelineResponse);

          // Suggest a version name based on pipeline name
          const currDate = new Date();
          setPipelineVersionName(
            pipelineResponse.display_name +
              '-version-at-' +
              currDate
                .toISOString()
                .toLowerCase()
                .replace(/:/g, '-'),
          );
        } catch (err) {
          logger.error('Error loading pipeline:', err);
        }
      }
    };

    loadPipelineData();
  }, [urlParser]);

  const buildInfo = props.buildInfo;

  return (
    <div className={classes(commonCss.page, padding(20, 'lr'))}>
      <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
        {/* Two subpages: one for creating version under existing pipeline and one for creating version under new pipeline */}
        <div className={classes(padding(10, 't'))}>Upload pipeline or pipeline version.</div>
        <div className={classes(commonCss.flex, padding(10, 'b'))}>
          <FormControlLabel
            id='createNewPipelineBtn'
            label='Create a new pipeline'
            checked={newPipeline === true}
            control={<Radio color='primary' data-testid='createNewPipelineBtn' />}
            onChange={() => {
              setCodeSourceUrl('');
              setNewPipeline(true);
              setPipelineDescription('');
              setPipelineName('');
              setPipelineDisplayName('');
              setPipelineVersionName('');
              setPipelineVersionDisplayName('');
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
              setPipelineVersionDescription('');
              setPipelineName('');
              setPipelineDisplayName('');
              setPipelineVersionName('');
              setPipelineVersionDisplayName('');
            }}
          />
        </div>

        {newPipeline === true && props.buildInfo?.apiServerMultiUser && (
          <PrivateSharedSelector
            onChange={val => {
              setIsPrivate(val);
            }}
          />
        )}

        {/* Pipeline name and help text for uploading new pipeline */}
        {newPipeline === true && (
          <>
            <div>Upload pipeline with the specified package.</div>
            <Input
              id='newPipelineName'
              value={pipelineName}
              required={true}
              label={
                'Pipeline Name' +
                (buildInfo?.pipelineStore === 'kubernetes' ? ' (Kubernetes object name)' : '')
              }
              variant='outlined'
              inputRef={pipelineNameRef}
              onChange={handleChange('pipelineName')}
              autoFocus={true}
            />
            {buildInfo?.pipelineStore === 'kubernetes' && (
              <Input
                id='newPipelineDisplayName'
                value={pipelineDisplayName}
                required={false}
                label='Pipeline Display Name'
                variant='outlined'
                inputRef={pipelineDisplayNameRef}
                onChange={handleChange('pipelineDisplayName')}
                autoFocus={true}
              />
            )}
            <Input
              id='pipelineDescription'
              value={pipelineDescription}
              required={false}
              label='Pipeline Description'
              variant='outlined'
              inputRef={pipelineDescriptionRef}
              onChange={handleChange('pipelineDescription')}
              autoFocus={true}
            />
          </>
        )}

        {/* Pipeline selector and help text for uploading new pipeline version */}
        {newPipeline === false && (
          <>
            <div>Upload pipeline version with the specified package.</div>
            {/* Select pipeline */}
            <Input
              value={pipelineDisplayName || pipelineName}
              required={true}
              label='Pipeline'
              disabled={true}
              variant='outlined'
              inputRef={pipelineNameRef}
              onChange={handleChange('pipelineName')}
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
              open={pipelineSelectorOpen}
              selectorDialog={css.selectorDialog}
              onClose={(confirmed, selectedPipeline?: V2beta1Pipeline) => {
                setUnconfirmedSelectedPipeline(selectedPipeline);
                pipelineSelectorClosed(confirmed);
              }}
              namespace={props.namespace}
              pipelineSelectorColumns={pipelineSelectorColumns}
            />

            {/* Set pipeline version name */}
            <Input
              id='pipelineVersionName'
              label={
                'Pipeline Version Name' +
                (buildInfo?.pipelineStore === 'kubernetes' ? ' (Kubernetes object name)' : '')
              }
              inputRef={pipelineVersionNameRef}
              required={true}
              onChange={handleChange('pipelineVersionName')}
              value={pipelineVersionName}
              autoFocus={true}
              variant='outlined'
            />
            {buildInfo?.pipelineStore === 'kubernetes' && (
              <Input
                id='pipelineVersionDisplayName'
                label='Pipeline Version Display Name'
                inputRef={pipelineVersionDisplayNameRef}
                required={false}
                onChange={handleChange('pipelineVersionDisplayName')}
                value={pipelineVersionDisplayName}
                autoFocus={true}
                variant='outlined'
              />
            )}
            <Input
              id='pipelineVersionDescription'
              value={pipelineVersionDescription}
              required={false}
              label='Pipeline Version Description'
              variant='outlined'
              onChange={handleChange('pipelineVersionDescription')}
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
          <div className={padding(10, 'b')}>
            Specify a publicly accessible package url and give the pipeline a unique name. The
            package url should be of one of these forms:
            <br />- a <strong>gcs bucket</strong> path, like:
            gs://your-bucket/path/to/my-pipeline.zip
            <br />- a <strong>tarball</strong> url (from github/gitlab), like:
            https://github.com/kubeflow/pipelines/archive/0.1.31.tar.gz
            <br />- a <strong>yaml file</strong> url, like:
            https://raw.githubusercontent.com/kubeflow/pipelines/0.1.31/samples/basic/sequential.yaml
            <br />- or a <strong>zip file</strong> that contains pipeline.yaml and component files.
          </div>
        )}

        <div className={classes(commonCss.flex, padding(10, 'b'))}>
          <div style={{ flex: '0 0 50px' }}>Package</div>
          <div style={{ flex: 1 }}>
            <Dropzone
              onDrop={onDrop}
              onDragEnter={() => setDropzoneActive(true)}
              onDragLeave={() => setDropzoneActive(false)}
              multiple={false}
              disabled={importMethod === ImportMethod.URL}
            >
              {({ getRootProps, getInputProps, isDragActive }) => (
                <div {...getRootProps()} className={css.dropOverlay}>
                  <input {...getInputProps()} />
                  <div
                    style={{
                      alignItems: 'center',
                      display: 'flex',
                      padding: '0 10px',
                    }}
                  >
                    <FormControlLabel
                      id='localPackageBtn'
                      label='Upload a file'
                      checked={importMethod === ImportMethod.LOCAL}
                      control={<Radio color='primary' />}
                      onChange={() => setImportMethod(ImportMethod.LOCAL)}
                    />
                    <Input
                      onChange={() => null}
                      value={fileName}
                      required={importMethod === ImportMethod.LOCAL}
                      label='Choose file'
                      variant='outlined'
                      disabled={importMethod === ImportMethod.URL}
                      // Find a better to align this input box with others
                      style={{
                        maxWidth: 2000,
                        width: 465,
                      }}
                      InputProps={{
                        classes: {
                          disabled: css.nonEditableInput,
                        },
                        endAdornment: (
                          <InputAdornment position='end'>
                            <Button
                              color='secondary'
                              disabled={importMethod === ImportMethod.URL}
                              onClick={() => {
                                if (pipelineNameRef.current) {
                                  pipelineNameRef.current.click();
                                }
                              }}
                              style={{ padding: '3px 5px', margin: 0, minWidth: 0 }}
                            >
                              Browse
                            </Button>
                          </InputAdornment>
                        ),
                        readOnly: true,
                        style: {
                          maxWidth: 2000,
                          width: 400,
                        },
                      }}
                    />
                  </div>
                </div>
              )}
            </Dropzone>
          </div>
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
            onChange={handleChange('packageUrl')}
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
          onChange={handleChange('codeSourceUrl')}
          required={false}
          value={codeSourceUrl}
          variant='outlined'
        />

        {/* Create pipeline or pipeline version */}
        <div className={commonCss.flex}>
          <BusyButton
            id='createNewPipelineOrVersionBtn'
            disabled={!!validationError}
            busy={isbeingCreated}
            className={commonCss.buttonAction}
            title={'Create'}
            onClick={create}
          />
          <Button
            id='cancelNewPipelineOrVersionBtn'
            onClick={() => props.navigate(RoutePage.PIPELINES)}
          >
            Cancel
          </Button>
          <div className={css.errorMessage}>{validationError}</div>
        </div>
      </div>
    </div>
  );
}

const EnhancedNewPipelineVersion: React.FC<PageProps> = props => {
  const buildInfo = React.useContext(BuildInfoContext);
  const namespace = React.useContext(NamespaceContext);

  return <NewPipelineVersion {...props} buildInfo={buildInfo} namespace={namespace} />;
};

export default EnhancedNewPipelineVersion;
