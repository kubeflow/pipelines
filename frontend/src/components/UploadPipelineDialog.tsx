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
import DropzoneArea, { DropzoneAreaHandle } from '../atoms/DropzoneArea';
import { ExternalLink } from '../atoms/ExternalLink';
import PrivateSharedSelector from './PrivateSharedSelector';
import { BuildInfoContext } from 'src/lib/BuildInfo';

import { Button } from './ui/button';
import { TextField } from './ui/text-field';
import { ModalDialog } from './ui/dialog';
import './modernization/PipelineForms.css';

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
    isPrivatePipeline: boolean,
    description?: string,
  ) => Promise<boolean>;
}

interface UploadPipelineDialogState {
  busy: boolean;
  dropzoneActive: boolean;
  file: File | null;
  fileError: string;
  fileName: string;
  fileUrl: string;
  importMethod: ImportMethod;
  uploadPipelineDescription: string;
  uploadPipelineName: string;
  isPrivatePipeline: boolean;
}

export const PIPELINE_PACKAGE_ACCEPT = {
  'application/yaml': ['.yaml', '.yml'],
  'application/zip': ['.zip'],
  'application/gzip': ['.tar.gz'],
};

export const PIPELINE_PACKAGE_REJECT_MESSAGE =
  'Invalid file type. Supported formats: .yaml, .yml, .zip, .tar.gz';

/**
 * Secondary validator for react-dropzone's {@link useDropzone} hook.
 *
 * The accept map must include `application/gzip` so the native file picker
 * (File System Access API) correctly recognises `.tar.gz` files. However,
 * plain `.gz` files share the same MIME type and pass the built-in MIME
 * check. This validator rejects them before they reach `onDropAccepted`.
 */
export function pipelinePackageValidator(file: File): { code: string; message: string } | null {
  const name = file.name.toLowerCase();
  if (name.endsWith('.gz') && !name.endsWith('.tar.gz')) {
    return { code: 'invalid-extension', message: PIPELINE_PACKAGE_REJECT_MESSAGE };
  }
  if (name.endsWith('.tgz')) {
    return { code: 'invalid-extension', message: PIPELINE_PACKAGE_REJECT_MESSAGE };
  }
  return null;
}

class UploadPipelineDialog extends React.Component<
  UploadPipelineDialogProps,
  UploadPipelineDialogState
> {
  static contextType = BuildInfoContext;
  declare context: React.ContextType<typeof BuildInfoContext>;
  private _dropzoneRef = React.createRef<DropzoneAreaHandle>();

  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      dropzoneActive: false,
      file: null,
      fileError: '',
      fileName: '',
      fileUrl: '',
      importMethod: ImportMethod.LOCAL,
      uploadPipelineDescription: '',
      uploadPipelineName: '',
      isPrivatePipeline: true,
    };
  }

  public render(): React.JSX.Element | null {
    const {
      dropzoneActive,
      file,
      fileError,
      fileName,
      fileUrl,
      importMethod,
      uploadPipelineName,
      busy,
    } = this.state;
    if (!this.props.open) return null;
    return (
      <ModalDialog
        open
        title='Upload and name your pipeline'
        onClose={() => this._uploadDialogClosed(false)}
        actions={
          <>
            <Button
              variant='secondary'
              id='cancelUploadBtn'
              onClick={() => this._uploadDialogClosed(false)}
            >
              Cancel
            </Button>
            <Button
              id='confirmUploadBtn'
              onClick={() => this._uploadDialogClosed(true)}
              aria-busy={busy}
              disabled={
                busy ||
                !uploadPipelineName ||
                (importMethod === ImportMethod.LOCAL ? !file : !fileUrl)
              }
            >
              Upload
            </Button>
          </>
        }
      >
        <div className='kfp-pipeline-upload'>
          {this.context?.apiServerMultiUser && (
            <PrivateSharedSelector
              value={this.state.isPrivatePipeline}
              onChange={(isPrivatePipeline) => this.setState({ isPrivatePipeline })}
            />
          )}
          <p className='kfp-pipeline-form-hint'>
            Upload a pipeline package file from your computer or import one using a URL.
          </p>
          <fieldset className='kfp-pipeline-choice'>
            <legend>Pipeline package source</legend>
            <label>
              <input
                id='uploadLocalFileBtn'
                name='upload-package-source'
                type='radio'
                checked={importMethod === ImportMethod.LOCAL}
                onChange={() => this.setState({ importMethod: ImportMethod.LOCAL, fileError: '' })}
              />
              Upload a file
            </label>
            <label>
              <input
                id='uploadFromUrlBtn'
                name='upload-package-source'
                type='radio'
                checked={importMethod === ImportMethod.URL}
                onChange={() => this.setState({ importMethod: ImportMethod.URL, fileError: '' })}
              />
              Import by URL
            </label>
          </fieldset>
          <DocumentationCompilePipeline />
          {importMethod === ImportMethod.LOCAL ? (
            <div className='kfp-pipeline-dropzone'>
              <DropzoneArea
                id='dropZone'
                data-testid='upload-pipeline-dropzone'
                aria-label='Pipeline package drop zone'
                onDrop={this._onDrop.bind(this)}
                onDropRejected={this._onDropRejected.bind(this)}
                onDragEnter={this._onDropzoneDragEnter.bind(this)}
                onDragLeave={this._onDropzoneDragLeave.bind(this)}
                accept={PIPELINE_PACKAGE_ACCEPT}
                validator={pipelinePackageValidator}
                ref={this._dropzoneRef}
                inputProps={{ tabIndex: -1 }}
              >
                {dropzoneActive && <div className='kfp-pipeline-drop-overlay'>Drop files…</div>}
                <p className='kfp-pipeline-form-hint'>You can also drag and drop the file here.</p>
                <TextField
                  value={fileName}
                  label='File'
                  required
                  readOnly
                  error={fileError}
                  trailingContent={
                    <Button variant='secondary' onClick={() => this._dropzoneRef.current?.open()}>
                      Choose file
                    </Button>
                  }
                />
              </DropzoneArea>
            </div>
          ) : (
            <TextField
              onChange={this.handleChange('fileUrl')}
              value={fileUrl}
              required
              label='URL'
              hint='URL must be publicly accessible.'
            />
          )}
          <TextField
            id='uploadFileName'
            label='Pipeline name'
            onChange={this.handleChange('uploadPipelineName')}
            required
            autoFocus
            value={uploadPipelineName}
          />
        </div>
      </ModalDialog>
    );
  }

  public handleChange =
    (name: string) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      this.setState({
        [name]: event.target.value,
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
      fileError: '',
      file: files[0],
      fileName: files[0].name,
      uploadPipelineName: files[0].name.split('.')[0],
    });
  }

  private _onDropRejected(): void {
    this.setState({
      dropzoneActive: false,
      file: null,
      fileName: '',
      uploadPipelineName: '',
      fileError: PIPELINE_PACKAGE_REJECT_MESSAGE,
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
        this.state.isPrivatePipeline,
        this.state.uploadPipelineDescription,
      );
      if (success) {
        this.setState({
          busy: false,
          dropzoneActive: false,
          file: null,
          fileError: '',
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

export default UploadPipelineDialog;

export const DocumentationCompilePipeline: React.FC = () => (
  <div className='kfp-pipeline-form-hint'>
    For expected file format, refer to{' '}
    <ExternalLink href='https://www.kubeflow.org/docs/components/pipelines/v2/compile-a-pipeline/'>
      Compile Pipeline Documentation
    </ExternalLink>
    .
  </div>
);
