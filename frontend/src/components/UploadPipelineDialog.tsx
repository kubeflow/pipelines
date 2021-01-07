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
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogTitle from '@material-ui/core/DialogTitle';
import Dropzone from 'react-dropzone';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Input from '../atoms/Input';
import InputAdornment from '@material-ui/core/InputAdornment';
import Radio from '@material-ui/core/Radio';
import { TextFieldProps } from '@material-ui/core/TextField';
import { padding, commonCss, zIndex, color } from '../Css';
import { stylesheet, classes } from 'typestyle';
import { ExternalLink } from '../atoms/ExternalLink';
import { TFunction } from 'i18next';
import { withTranslation, useTranslation } from 'react-i18next';

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
  root: {
    width: 500,
  },
});

export enum ImportMethod {
  LOCAL = 'local',
  URL = 'url',
}

interface UploadPipelineDialogProps {
  open: boolean;
  onClose: (
    confirmed: boolean,
    name: string,
    file: File | null,
    url: string,
    method: ImportMethod,
    description?: string,
  ) => Promise<boolean>;
  t: TFunction;
}

interface UploadPipelineDialogState {
  busy: boolean;
  dropzoneActive: boolean;
  file: File | null;
  fileName: string;
  fileUrl: string;
  importMethod: ImportMethod;
  uploadPipelineDescription: string;
  uploadPipelineName: string;
}

class UploadPipelineDialog extends React.Component<
  UploadPipelineDialogProps,
  UploadPipelineDialogState
> {
  private _dropzoneRef = React.createRef<Dropzone & HTMLDivElement>();

  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      dropzoneActive: false,
      file: null,
      fileName: '',
      fileUrl: '',
      importMethod: ImportMethod.LOCAL,
      uploadPipelineDescription: '',
      uploadPipelineName: '',
    };
  }

  public render(): JSX.Element {
    const {
      dropzoneActive,
      file,
      fileName,
      fileUrl,
      importMethod,
      uploadPipelineName,
      busy,
    } = this.state;
    const { t } = this.props;

    return (
      <Dialog
        id='uploadDialog'
        onClose={() => this._uploadDialogClosed(false)}
        open={this.props.open}
        classes={{ paper: css.root }}
      >
        <DialogTitle>{t('uploadNamePipeline')}</DialogTitle>

        <div className={padding(20, 'lr')}>
          <div className={classes(commonCss.flex, padding(10, 'b'))}>
            <FormControlLabel
              id='uploadLocalFileBtn'
              label={t('common:uploadFile')}
              checked={importMethod === ImportMethod.LOCAL}
              control={<Radio color='primary' />}
              onChange={() => this.setState({ importMethod: ImportMethod.LOCAL })}
            />
            <FormControlLabel
              id='uploadFromUrlBtn'
              label={t('importByUrl')}
              checked={importMethod === ImportMethod.URL}
              control={<Radio color='primary' />}
              onChange={() => this.setState({ importMethod: ImportMethod.URL })}
            />
          </div>

          {importMethod === ImportMethod.LOCAL && (
            <React.Fragment>
              <Dropzone
                id='dropZone'
                disableClick={true}
                onDrop={this._onDrop.bind(this)}
                onDragEnter={this._onDropzoneDragEnter.bind(this)}
                onDragLeave={this._onDropzoneDragLeave.bind(this)}
                style={{ position: 'relative' }}
                ref={this._dropzoneRef}
                inputProps={{ tabIndex: -1 }}
              >
                {dropzoneActive && <div className={css.dropOverlay}>{t('dropFiles')}</div>}

                <div className={padding(10, 'b')}>
                  {t('uploadPackagePipelineFileInstructions')}
                  <br />
                  {t('dragAndDropFile')}
                </div>
                <DocumentationCompilePipeline />
                <Input
                  onChange={this.handleChange('fileName')}
                  value={fileName}
                  required={true}
                  label={t('common:file')}
                  variant='outlined'
                  InputProps={{
                    endAdornment: (
                      <InputAdornment position='end'>
                        <Button
                          color='secondary'
                          onClick={() => this._dropzoneRef.current!.open()}
                          style={{ padding: '3px 5px', margin: 0, whiteSpace: 'nowrap' }}
                        >
                          {t('common:chooseFile')}
                        </Button>
                      </InputAdornment>
                    ),
                    readOnly: true,
                  }}
                />
              </Dropzone>
            </React.Fragment>
          )}

          {importMethod === ImportMethod.URL && (
            <React.Fragment>
              <div className={padding(10, 'b')}>{t('urlPublic')}</div>
              <DocumentationCompilePipeline />
              <Input
                onChange={this.handleChange('fileUrl')}
                value={fileUrl}
                required={true}
                label={t('common:url')}
                variant='outlined'
              />
            </React.Fragment>
          )}

          <Input
            id='uploadFileName'
            label={t('common:pipelineName')}
            onChange={this.handleChange('uploadPipelineName')}
            required={true}
            value={uploadPipelineName}
            variant='outlined'
          />
        </div>

        {/* <Input label='Pipeline description'
          onChange={this.handleChange('uploadPipelineDescription')}
          value={uploadPipelineDescription} multiline={true} variant='outlined' /> */}

        <DialogActions>
          <Button id='cancelUploadBtn' onClick={() => this._uploadDialogClosed.bind(this)(false)}>
            {t('common:cancel')}
          </Button>
          <BusyButton
            id='confirmUploadBtn'
            onClick={() => this._uploadDialogClosed.bind(this)(true)}
            title={t('common:upload')}
            busy={busy}
            disabled={
              !uploadPipelineName || (importMethod === ImportMethod.LOCAL ? !file : !fileUrl)
            }
          />
        </DialogActions>
      </Dialog>
    );
  }

  public handleChange = (name: string) => (event: any) => {
    this.setState({
      [name]: (event.target as TextFieldProps).value,
    } as any);
  };

  private _onDropzoneDragEnter(): void {
    this.setState({ dropzoneActive: true });
  }

  private _onDropzoneDragLeave(): void {
    this.setState({ dropzoneActive: false });
  }

  private _onDrop(files: File[]): void {
    this.setState({
      dropzoneActive: false,
      file: files[0],
      fileName: files[0].name,
      // Suggest all characters left of first . as pipeline name
      uploadPipelineName: files[0].name.split('.')[0],
    });
  }

  private _uploadDialogClosed(confirmed: boolean): void {
    this.setState({ busy: true }, async () => {
      const success = await this.props.onClose(
        confirmed,
        this.state.uploadPipelineName,
        this.state.file,
        this.state.fileUrl.trim(),
        this.state.importMethod,
        this.state.uploadPipelineDescription,
      );
      if (success) {
        this.setState({
          busy: false,
          dropzoneActive: false,
          file: null,
          fileName: '',
          fileUrl: '',
          importMethod: ImportMethod.LOCAL,
          uploadPipelineDescription: '',
          uploadPipelineName: '',
        });
      } else {
        this.setState({ busy: false });
      }
    });
  }
}

export default withTranslation(['pipelines', 'common'])(UploadPipelineDialog);

const DocumentationCompilePipeline: React.FC = () => {
  const { t } = useTranslation('pipelines');
  return (
    <div className={padding(10, 'b')}>
      {t('expectedFileFormat')}{' '}
      <ExternalLink href='https://www.kubeflow.org/docs/pipelines/sdk/build-component/#compile-the-pipeline'>
        {t('compilePipelineDoc')}
      </ExternalLink>
      .
    </div>
  );
};
