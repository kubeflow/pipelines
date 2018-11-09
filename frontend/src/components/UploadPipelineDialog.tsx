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
import Input from '../atoms/Input';
import { stylesheet } from 'typestyle';
import { padding } from '../Css';
import InputAdornment from '@material-ui/core/InputAdornment';
import { TextFieldProps } from '@material-ui/core/TextField';

const css = stylesheet({
  dropOverlay: {
    backgroundColor: '#eee',
    border: '2px dashed #aaa',
    bottom: 0,
    left: 0,
    margin: 3,
    padding: '2.5em 0',
    position: 'absolute',
    right: 0,
    textAlign: 'center',
    top: 0,
    zIndex: 1,
  },
  root: {
    minWidth: 500,
  },
});

interface UploadPipelineDialogProps {
  open: boolean;
  onClose: (name: string, file: File | null, description?: string) => Promise<boolean>;
}

interface UploadPipelineDialogState {
  busy: boolean;
  dropzoneActive: boolean;
  file: File | null;
  fileName: string;
  uploadPipelineDescription: string;
  uploadPipelineName: string;
}

class UploadPipelineDialog extends React.Component<UploadPipelineDialogProps, UploadPipelineDialogState> {
  private _dropzoneRef = React.createRef<Dropzone & HTMLDivElement>();

  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      dropzoneActive: false,
      file: null,
      fileName: '',
      uploadPipelineDescription: '',
      uploadPipelineName: '',
    };
  }

  public render(): JSX.Element {
    const { dropzoneActive, file, uploadPipelineName, busy } = this.state;
    return (
      <Dialog id='uploadDialog' onClose={() => this._uploadDialogClosed(false)}
        open={this.props.open} classes={{ paper: css.root }}>
        <DialogTitle>Upload and name your pipeline</DialogTitle>
        <Dropzone id='dropZone' disableClick={true} className={padding(20, 'lr')}
          onDrop={this._onDrop.bind(this)} onDragEnter={this._onDropzoneDragEnter.bind(this)}
          onDragLeave={this._onDropzoneDragLeave.bind(this)} style={{ position: 'relative' }}
          ref={this._dropzoneRef} inputProps={{ tabIndex: -1 }}>

          {dropzoneActive && <div className={css.dropOverlay}>Drop files..</div>}

          <div className={padding(10, 'b')}>
            Choose a pipeline package file from your computer, and give the pipeline a unique name.
            <br />
            You can also drag and drop the file here.
          </div>
          <Input field='fileName' instance={this} required={true} label='File'
            InputProps={{
              endAdornment: (
                <InputAdornment position='end'>
                  <Button color='secondary' onClick={() => this._dropzoneRef.current!.open()}
                    style={{ padding: '3px 5px', margin: 0 }}>
                    Choose file
               </Button>
                </InputAdornment>
              ),
              readOnly: true,
            }} />

          <Input id='uploadFileName' label='Pipeline name' instance={this} required={true}
            field='uploadPipelineName' />

          {/* <Input label='Pipeline description' instance={this} field='uploadPipelineDescription'
            multiline={true} height='auto' /> */}
        </Dropzone>

        <DialogActions>
          <BusyButton id='confirmUploadBtn' onClick={() => this._uploadDialogClosed.bind(this)(true)}
            title='Upload' busy={busy} disabled={!file || !uploadPipelineName} />
          <Button id='cancelUploadBtn' onClick={() => this._uploadDialogClosed.bind(this)(false)}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    );
  }

  public handleChange = (name: string) => (event: any) => {
    this.setState({
      [name]: (event.target as TextFieldProps).value,
    } as any);
  }

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
      uploadPipelineName: files[0].name,
    });
  }

  private _uploadDialogClosed(confirmed: boolean): void {
    const file = confirmed ? this.state.file : null;
    this.setState({ busy: true }, async () => {
      const success = await this.props.onClose(
        this.state.uploadPipelineName, file, this.state.uploadPipelineDescription);
      if (success) {
        this.setState({
          busy: false,
          dropzoneActive: false,
          file: null,
          uploadPipelineDescription: '',
          uploadPipelineName: '',
        });
      } else {
        this.setState({
          busy: false,
        });
      }
    });
  }
}

export default UploadPipelineDialog;
