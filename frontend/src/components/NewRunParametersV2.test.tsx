/*
 * Copyright 2022 The Kubeflow Authors
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

import 'jest';
import React from 'react';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { ParameterType_ParameterTypeEnum } from 'src/generated/pipeline_spec/pipeline_spec';
import NewRunParametersV2 from 'src/components/NewRunParametersV2';
import { inputConverter } from 'src/components/NewRunParametersV2';

testBestPractices();

describe('NewRunParametersV2', () => {
  it('shows parameters', () => {
    render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{
            strParam: {
              parameterType: ParameterType_ParameterTypeEnum.STRING,
              defaultValue: 'string value',
            },
            boolParam: {
              parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
              defaultValue: true,
            },
            intParam: {
              parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
              defaultValue: 123,
            },
          }}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    expect(screen.getByText('Run parameters'));
    expect(screen.getByText('Specify parameters required by the pipeline'));
    expect(screen.getByText('strParam - STRING'));
    expect(screen.getByDisplayValue('string value'));
    expect(screen.getByText('boolParam - BOOLEAN'));
    expect(screen.getByDisplayValue('true'));
    expect(screen.getByText('intParam - NUMBER_INTEGER'));
    expect(screen.getByDisplayValue('123'));
  });

  it('edits parameters', () => {
    render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{
            strParam: {
              parameterType: ParameterType_ParameterTypeEnum.STRING,
              defaultValue: 'string value',
            },
            intParam: {
              parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
              defaultValue: 123,
            },
            boolParam: {
              parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
              defaultValue: true,
            },
          }}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    const strParam = screen.getByDisplayValue('string value');
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(strParam.closest('input').value).toEqual('new string');

    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: 999 } });
    expect(intParam.closest('input').value).toEqual('999');

    const boolParam = screen.getByDisplayValue('true');
    fireEvent.change(boolParam, { target: { value: false } });
    expect(boolParam.closest('input').value).toEqual('false');
  });

  it('convert string-type user input to real type', () => {
    expect(inputConverter('string value', ParameterType_ParameterTypeEnum.STRING)).toEqual(
      'string value',
    );
    expect(inputConverter('True', ParameterType_ParameterTypeEnum.BOOLEAN)).toEqual(true);
    expect(inputConverter('123', ParameterType_ParameterTypeEnum.NUMBER_INTEGER)).toEqual(123);
    expect(inputConverter('True', ParameterType_ParameterTypeEnum.STRING)).toEqual('True');
    expect(inputConverter('123', ParameterType_ParameterTypeEnum.STRING)).toEqual('123');
    expect(inputConverter('true', ParameterType_ParameterTypeEnum.BOOLEAN)).toEqual(null);
  });

  it('test convertInput function for LIST type', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };

    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const listParam = screen.getByLabelText('listParam - LIST');
    fireEvent.change(listParam, {target: {value : '[4,5,6]'}});
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: [4,5,6],
    })
  });

  it('test convertInput function for STRUCT type', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };

    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const listParam = screen.getByLabelText('structParam - STRUCT');
    fireEvent.change(listParam, {target: {value : '{A:1'}});
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: [4,5,6],
    })
  });

  it('does not display any text fields if there are no parameters', () => {
    const { container } = render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{}}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    expect(container.querySelector('input').type).toEqual('checkbox');
  });
});
