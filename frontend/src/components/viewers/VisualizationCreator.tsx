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
import 'brace';
import BusyButton from '../../atoms/BusyButton';
import FormControl from '@material-ui/core/FormControl';
import Input from '../../atoms/Input';
import InputLabel from '@material-ui/core/InputLabel';
import MenuItem from '@material-ui/core/MenuItem';
import Select from '@material-ui/core/Select';
import Editor from '../Editor';
import { ViewerConfig } from './Viewer';
import { ApiVisualizationType } from '../../apis/visualization';
import 'brace/ext/language_tools';
import 'brace/mode/json';
import 'brace/mode/python';
import 'brace/theme/github';
import Button from '@material-ui/core/Button';
import { TFunction } from 'i18next';

export interface VisualizationCreatorConfig extends ViewerConfig {
  allowCustomVisualizations?: boolean;
  // Whether there is currently a visualization being generated or not.
  isBusy?: boolean;
  // Function called to generate a visualization.
  onGenerate?: (visualizationArguments: string, source: string, type: ApiVisualizationType) => void;
  // Facilitate testing by not collapsing by default.
  collapsedInitially?: boolean;
}

interface VisualizationCreatorProps {
  configs: VisualizationCreatorConfig[];
  maxWidth?: number;
  t: TFunction;
}

interface VisualizationCreatorState {
  expanded: boolean;
  // arguments is expected to be a JSON object in string form.
  arguments: string;
  code: string;
  source: string;
  selectedType?: ApiVisualizationType;
}

class VisualizationCreator extends React.Component<
  VisualizationCreatorProps,
  VisualizationCreatorState
> {
  public state: VisualizationCreatorState = {
    expanded: !this.props.configs[0]?.collapsedInitially,
    arguments: '',
    code: '',
    source: '',
  };

  public render(): JSX.Element | null {
    const { configs } = this.props;
    const config = configs[0];
    const { arguments: _arguments, code, source, selectedType } = this.state;

    if (!config) {
      return null;
    }

    const { allowCustomVisualizations = false, isBusy = false, onGenerate } = config;

    // Only allow a visualization to be generated if one is not already being
    // generated (as indicated by the isBusy tag), and if there is an source
    // provided, and a visualization type is selected, and a onGenerate function
    // is provided.
    const hasSourceAndSelectedType = source.length > 0 && !!selectedType;
    const isCustomTypeAndHasCode = selectedType === ApiVisualizationType.CUSTOM && code.length > 0;
    const canGenerate =
      !isBusy && !!onGenerate && (hasSourceAndSelectedType || isCustomTypeAndHasCode);

    const argumentsPlaceholder = this.getArgumentPlaceholderForType(selectedType);

    if (!this.state.expanded) {
      return (
        <Button variant='text' onClick={this.handleExpansion}>
          create visualizations manually
        </Button>
      );
    }

    return (
      <div
        style={{
          width: this.props.maxWidth || 600,
        }}
      >
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
            {this.getAvailableTypes(allowCustomVisualizations).map((key: string) => (
              <MenuItem key={key} value={ApiVisualizationType[key]}>
                {ApiVisualizationType[key]}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <Input
          label='Source'
          variant={'outlined'}
          value={source}
          disabled={isBusy}
          placeholder='File path or path pattern of data within GCS.'
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            this.setState({ source: e.target.value })
          }
        />
        {selectedType === ApiVisualizationType.CUSTOM && (
          <div>
            <InputLabel>Custom Visualization Code</InputLabel>
            <Editor
              placeholder='Python code that will be run to generate visualization.<br><br>To access the source value (if provided), reference the variable "source".<br>To access any provided arguments, reference the variable "variables" (it is a dict object).'
              width='100%'
              height='175px'
              mode='python'
              theme='github'
              value={code}
              onChange={(value: string) => this.setState({ code: value })}
              editorProps={{ $blockScrolling: true }}
              enableLiveAutocompletion={true}
              enableBasicAutocompletion={true}
              highlightActiveLine={true}
              showGutter={true}
            />
          </div>
        )}
        {!!selectedType && (
          <div>
            <InputLabel>Arguments (Optional)</InputLabel>
            <Editor
              placeholder={argumentsPlaceholder}
              height={`${argumentsPlaceholder.split('<br>').length * 14}px`}
              width='100%'
              mode='json'
              theme='github'
              value={_arguments}
              onChange={(value: string) => this.setState({ arguments: value })}
              editorProps={{ $blockScrolling: true }}
              highlightActiveLine={true}
              showGutter={true}
            />
          </div>
        )}
        <BusyButton
          title='Generate Visualization'
          busy={isBusy}
          disabled={!canGenerate}
          onClick={() => {
            if (onGenerate && selectedType) {
              const specifiedArguments: any = JSON.parse(_arguments || '{}');
              if (selectedType === ApiVisualizationType.CUSTOM) {
                specifiedArguments.code = code.split('\n');
              }
              onGenerate(JSON.stringify(specifiedArguments), source, selectedType);
            }
          }}
        />
      </div>
    );
  }

  /*
    Due to the swagger API definition generation, enum value that include
    an _ (underscore) remove all _ from the enum key. Additionally, due to the
    manner in which TypeScript is compiled to Javascript, enums are duplicated
    iff they included an _ in the proto file. This filters out those duplicate
    keys that are generated by the complication from TypeScript to JavaScript.

    For example:
    export enum ApiVisualizationType {
      ROCCURVE = <any> 'ROC_CURVE'
    }

    Object.keys(ApiVisualizationType) = ['ROCCURVE', 'ROC_CURVE'];

    Additional details can be found here:
    https://www.typescriptlang.org/play/#code/KYOwrgtgBAggDgSwGoIM5gIYBsEC8MAuCA9iACoCecwUA3gLABQUUASgPIDCnAqq0gFEoAXigAeDCAoA+AOQdOAfV78BsgDRMAvkyYBjUqmJZgAOizEA5gAp4yNJhz4ipStQCUAbiA
  */
  private getAvailableTypes(allowCustomVisualizations: boolean): string[] {
    return Object.keys(ApiVisualizationType)
      .map((key: string) => key.replace('_', ''))
      .filter((key: string, i: number, arr: string[]) => {
        const isDuplicate = arr.indexOf(key) !== i;
        const isCustom = key === 'CUSTOM';
        const isTFMA = key === 'TFMA';
        return !isDuplicate && (allowCustomVisualizations || !isCustom) && !isTFMA;
      });
  }

  private getArgumentPlaceholderForType(type: ApiVisualizationType | undefined): string {
    let placeholder = 'Arguments, provided as JSON, to be used during visualization generation.';
    switch (type) {
      case ApiVisualizationType.ROCCURVE:
        // These arguments are not yet used as the ROC curve visualization is
        // still based on the Kubeflow Pipelines component.
        // placeholder = `{
        // \t"y_true": array,
        // \t"y_score": array,
        // \t"pos_label": number | string | null,
        // \t"sample_weight": array | null,
        // \t"drop_intermediate": boolean | null,
        // \t"is_generated": boolean | null,
        // }`;
        placeholder = `{
        \t"is_generated": boolean | null,
        \t"target_lambda": string | null,
        \t"trueclass": string | null,
        \t"true_score_column": string | null
        }`;
        break;
      case ApiVisualizationType.TFDV:
        placeholder = '{}';
        break;
      case ApiVisualizationType.TFMA:
        placeholder = `{
        \t"slicing_column: string | null
        }`;
        break;
      case ApiVisualizationType.TABLE:
        placeholder = '{\n\t"headers": array\n}';
        break;
      case ApiVisualizationType.CUSTOM:
        placeholder = '{\n\t"key": any\n}';
        break;
    }
    return (
      placeholder
        // Replaces newline escape character with HTML break so placeholder can
        // support multiple lines.
        // eslint-disable-next-line no-control-regex
        .replace(new RegExp('\n', 'g'), '<br>')
        // Replaces tab escape character with 4 blank spaces so placeholder can
        // support indentation.
        // eslint-disable-next-line no-control-regex
        .replace(new RegExp('\t', 'g'), '&nbsp&nbsp&nbsp&nbsp')
    );
  }

  private handleExpansion = () => {
    this.setState({
      expanded: true,
    });
  };
}

export default VisualizationCreator;
