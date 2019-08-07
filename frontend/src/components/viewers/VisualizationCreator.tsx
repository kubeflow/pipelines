/*
 * Copyright 2019 Google LLC
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
import BusyButton from '../../atoms/BusyButton';
import FormControl from '@material-ui/core/FormControl';
import Input from '../../atoms/Input';
import InputLabel from '@material-ui/core/InputLabel';
import MenuItem from '@material-ui/core/MenuItem';
import Select from '@material-ui/core/Select';
import Viewer, { ViewerConfig } from './Viewer';
import { ApiVisualizationType } from '../../apis/visualization';


export interface VisualizationCreatorConfig extends ViewerConfig {
  // Whether there is currently a visualization being generated or not.
  isBusy?: boolean;
  // Function called to generate a visualization.
  onGenerate?: (visualizationArguments: string, source: string, type: ApiVisualizationType) => void;
}

interface VisualizationCreatorProps {
  configs: VisualizationCreatorConfig[];
  maxWidth?: number;
}

interface VisualizationCreatorState {
  // arguments is expected to be a JSON object in string form.
  arguments: string;
  source: string;
  selectedType?: ApiVisualizationType;
}

class VisualizationCreator extends Viewer<VisualizationCreatorProps, VisualizationCreatorState> {
  constructor(props: VisualizationCreatorProps) {
    super(props);
    this.state = {
      arguments: '',
      source: '',
    };
  }

  public getDisplayName(): string {
    return 'Visualization Creator';
  }

  public render(): JSX.Element | null {
    const { configs } = this.props;
    const config = configs[0];
    const { arguments: _arguments, source, selectedType } = this.state;

    if (!config) {
      return null;
    }

    const { isBusy = false, onGenerate } = config;

    // Only allow a visualization to be generated if one is not already being
    // generated (as indicated by the isBusy tag), and if there is an source
    // provided, and a visualization type is selected, and a onGenerate function
    // is provided.
    const canGenerate = !isBusy &&
      !!source.length &&
      !!selectedType &&
      !!onGenerate;

    return <div
      style={{
        width: this.props.maxWidth || 600
      }}>
      <FormControl style={{ width: '100%' }}>
        <InputLabel htmlFor='visualization-type-selector'>Type</InputLabel>
        <Select
          value={selectedType}
          inputProps={{
            id: 'visualization-type-selector',
            name: 'Visualization Type',
          }}
          style={{
            minHeight: 60,
            width: '100%',
          }}
          onChange={(e: React.ChangeEvent<{ name?: string; value: unknown }>) => {
            this.setState({ selectedType: e.target.value as ApiVisualizationType });
          }}
          disabled={isBusy}
        >
          {/*
          Due to the swagger API definition generation, getting the keys within
          ApiVisualizationType also provides the values for the keys. This
          filters out those values and only shows the actual keys as a
          visualization type.

          For example:
          export enum ApiVisualizationType {
            CURVE = <any> 'ROC_CURVE'
          }

          Object.keys(ApiVisualizationType) = ['CURVE', 'ROC_CURVE'];

          This occurs due to the dictation of the any type.
        */}
          {Object.keys(ApiVisualizationType)
            .filter((_, i: number) => i % 2 === 0)
            .map((key: string) => (
              <MenuItem key={key} value={ApiVisualizationType[key]}>{key}</MenuItem>
            ))}
        </Select>
      </FormControl>

      <Input label='Source' variant={'outlined'} value={source} disabled={isBusy}
        placeholder='File path or path pattern of data within GCS.'
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => this.setState({ source: e.target.value })} />
      <Input label='Arguments (optional)' multiline={true} variant='outlined'
        value={_arguments} disabled={isBusy}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => this.setState({ arguments: e.target.value })} />
      <BusyButton title='Generate Visualization' busy={isBusy} disabled={!canGenerate}
        onClick={() => {
          if (onGenerate && selectedType) {
            onGenerate(_arguments, source, selectedType);
          }
        }} />
    </div>;
  }
}

export default VisualizationCreator;
